package ai

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/features/product"
	"github.com/naskung/api-Costbook/internal/helpers"
	"github.com/naskung/api-Costbook/internal/utils"
)

// เวลารอสูงสุดของงาน AI (เช็คราคาตลาดค้นเว็บหลายรอบ ใช้นานกว่าแชท)
const (
	marketTimeout   = 5 * time.Minute
	chatTimeout     = 3 * time.Minute
	keyCheckTimeout = 30 * time.Second
)

type Handler struct {
	repo      Repository
	shop      ShopReader
	cfg       Config
	now       func() time.Time
	newClaude func(apiKey string) Claude
	newGemini func(apiKey, model string) Gemini
	newOpenAI func(baseURL, apiKey, model string) OpenAI
	// newSearcher ตัวค้นเว็บของ Gemini ฟรี (Tavily) — test เปลี่ยนเป็นตัวปลอมได้
	newSearcher func(apiKey string) WebSearcher
}

func NewHandler(repo Repository, shop ShopReader, cfg Config) *Handler {
	return &Handler{
		repo: repo, shop: shop, cfg: cfg, now: time.Now,
		newClaude: NewClaude, newGemini: NewGemini, newOpenAI: NewOpenAI, newSearcher: NewTavily,
	}
}

// RegisterRoutes ต่อ route ของ AI เข้ากับกลุ่ม /api/v1 (ต้อง login ทุกเส้น)
func (h *Handler) RegisterRoutes(api gin.IRouter, authRequired gin.HandlerFunc) {
	api.GET("/ai/status", authRequired, h.Status)
	api.PUT("/ai/key", authRequired, h.SaveKey)
	api.DELETE("/ai/key", authRequired, h.DeleteKey)
	api.POST("/ai/models", authRequired, h.Models)
	api.POST("/ai/chat", authRequired, h.Chat)
	api.GET("/products/:id/market", authRequired, h.GetMarketCheck)
	api.POST("/products/:id/market", authRequired, h.CheckMarket)
}

// current AI ที่ใช้อยู่ตอนนี้ + สถานะ (assistant = nil เมื่อยังไม่มี AI ให้ใช้)
func (h *Handler) current(ctx context.Context) (Assistant, Status, error) {
	saved, err := h.savedAI(ctx)
	if err != nil {
		return nil, Status{}, err
	}
	status := pickProvider(saved, h.cfg)
	switch {
	case status.Provider == providerOpenAI:
		assistant := openAIAssistant{client: h.newOpenAI(saved.BaseURL, saved.APIKey, saved.Model)}
		if h.cfg.TavilyKey != "" {
			assistant.searcher = h.newSearcher(h.cfg.TavilyKey)
		}
		return assistant, status, nil
	case status.Provider == providerClaude && status.Source == sourceSettings:
		return claudeAssistant{claude: h.newClaude(saved.APIKey)}, status, nil
	case status.Provider == providerClaude:
		return claudeAssistant{claude: h.newClaude(h.cfg.ClaudeKey)}, status, nil
	case status.Provider == providerGemini:
		models := geminiModelsOf(status.Model, geminiFallbacksOf(h.cfg))
		newGemini := func(model string) Gemini { return h.newGemini(h.cfg.GeminiKey, model) }
		assistant := geminiAssistant{
			gemini:    newFallbackGemini(models, newGemini),
			search:    h.newGemini(h.cfg.GeminiKey, geminiSearchModelOf(h.cfg)),
			useGoogle: h.cfg.GeminiWebSearch,
		}
		if h.cfg.TavilyKey != "" {
			assistant.searcher = h.newSearcher(h.cfg.TavilyKey)
		}
		return assistant, status, nil
	}
	return nil, status, nil
}

// savedAI AI ที่ร้านตั้งไว้ (ถอดรหัส key แล้ว) — ถอดไม่ได้ (เช่น เปลี่ยน JWT_SECRET) ถือว่ายังไม่ได้ตั้ง ให้ร้านตั้งใหม่
func (h *Handler) savedAI(ctx context.Context) (aiSettings, error) {
	saved, err := h.repo.SavedAI(ctx)
	if err != nil || saved.Provider == "" {
		return aiSettings{}, err
	}
	settings := aiSettings{Provider: saved.Provider, BaseURL: saved.BaseURL, Model: saved.Model}
	if saved.EncryptedKey == "" {
		return settings, nil
	}
	settings.APIKey, err = utils.Decrypt(saved.EncryptedKey, h.cfg.Secret)
	if err != nil {
		slog.Warn("ถอดรหัส key ของ AI ไม่ได้ (เปลี่ยน JWT_SECRET?) ให้ตั้ง AI ใหม่ในหน้าตั้งค่า")
		return aiSettings{}, nil
	}
	return settings, nil
}

// Status GET /api/v1/ai/status — AI ที่ใช้อยู่ (Claude / AI แบบ OpenAI / Gemini ฟรี / ไม่มี) ค้นเว็บได้ไหม ตั้งเองไว้ไหม
func (h *Handler) Status(c *gin.Context) {
	_, status, err := h.current(c.Request.Context())
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, status)
}

// SaveKey PUT /api/v1/ai/key — ร้านเลือก AI เอง ลองต่อจริงก่อน ใช้ได้ถึงเก็บ (key เข้ารหัส)
//   - Claude: {"api_key": "sk-ant-..."} (provider ไม่ส่งก็ได้)
//   - AI แบบ OpenAI: {"provider": "openai", "base_url": "https://api.deepseek.com/v1", "api_key": "...", "model": "deepseek-chat"}
func (h *Handler) SaveKey(c *gin.Context) {
	var req SaveAIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.WriteError(c, http.StatusBadRequest, "invalid_json")
		return
	}
	req = req.normalized()
	if code := req.Validate(); code != "" {
		helpers.WriteError(c, http.StatusBadRequest, code)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), keyCheckTimeout)
	defer cancel()
	if !h.checkConnection(c, ctx, req) {
		return
	}

	saved := SavedAI{Provider: req.Provider, BaseURL: req.BaseURL, Model: req.Model}
	if req.Provider == providerClaude {
		saved.BaseURL, saved.Model = "", ""
	}
	var err error
	if req.APIKey != "" {
		saved.EncryptedKey, err = utils.Encrypt(req.APIKey, h.cfg.Secret)
	}
	if err == nil {
		err = h.repo.SaveAI(c.Request.Context(), saved)
	}
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	h.Status(c)
}

// checkConnection ลองต่อ AI ที่ร้านเลือกจริง (ไม่เสียเงิน) — ใช้ไม่ได้ → ตอบ error แล้วคืน false
func (h *Handler) checkConnection(c *gin.Context, ctx context.Context, req SaveAIRequest) bool {
	if req.Provider == providerOpenAI {
		return h.checkOpenAI(c, ctx, req)
	}
	return h.checkClaude(c, ctx, req.APIKey)
}

// checkClaude ลอง key กับ Claude (ขอรายชื่อรุ่น ไม่เสียเงิน) — ใช้ไม่ได้ → ตอบ error แล้วคืน false
func (h *Handler) checkClaude(c *gin.Context, ctx context.Context, key string) bool {
	err := h.newClaude(key).CheckKey(ctx)
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) && (apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden) {
		helpers.WriteError(c, http.StatusBadRequest, "ai_key_invalid")
		return false
	}
	if err != nil {
		writeAIError(c, err)
		return false
	}
	return true
}

// checkOpenAI ลองต่อ AI แบบ OpenAI (GET /models ไม่เสียเงิน): ต่อได้ + key ถูก + มีรุ่นที่เลือก
// บางเจ้าไม่มีรายชื่อรุ่นให้ (404) → ข้ามการเช็ครุ่น ไปรู้ตอนแชทแทน
func (h *Handler) checkOpenAI(c *gin.Context, ctx context.Context, req SaveAIRequest) bool {
	models, ok := h.listModels(c, ctx, req.BaseURL, req.APIKey)
	if !ok {
		return false
	}
	if len(models) > 0 && !slices.Contains(models, req.Model) {
		helpers.WriteError(c, http.StatusBadRequest, "ai_model_not_found")
		return false
	}
	return true
}

// listModels รายชื่อรุ่นของ AI แบบ OpenAI — ไม่มีรายชื่อให้ (404) = ว่าง / ต่อไม่ได้, key ผิด → ตอบ error แล้วคืน false
func (h *Handler) listModels(c *gin.Context, ctx context.Context, baseURL, apiKey string) ([]string, bool) {
	models, err := h.newOpenAI(baseURL, apiKey, "").Models(ctx)
	var apiErr *OpenAIError
	switch {
	case err == nil:
		slices.Sort(models)
		return models, true
	case errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound:
		return []string{}, true
	case errors.As(err, &apiErr) && (apiErr.Status == http.StatusUnauthorized || apiErr.Status == http.StatusForbidden):
		helpers.WriteError(c, http.StatusBadRequest, "ai_key_invalid")
	case errors.As(err, &apiErr), errors.Is(err, context.DeadlineExceeded):
		writeAIError(c, err)
	default:
		writeAICode(c, http.StatusBadRequest, "ai_connect_failed", err.Error())
	}
	return nil, false
}

// Models POST /api/v1/ai/models {"base_url", "api_key"} — รายชื่อรุ่นของ AI แบบ OpenAI ไว้ให้ร้านเลือก (ไม่เสียเงิน)
func (h *Handler) Models(c *gin.Context) {
	var req SaveAIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.WriteError(c, http.StatusBadRequest, "invalid_json")
		return
	}
	req = req.normalized()
	if !validBaseURL(req.BaseURL) {
		helpers.WriteError(c, http.StatusBadRequest, "ai_base_url_invalid")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), keyCheckTimeout)
	defer cancel()
	models, ok := h.listModels(c, ctx, req.BaseURL, req.APIKey)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"models": models})
}

// DeleteKey DELETE /api/v1/ai/key — ลบ AI ที่ร้านตั้งไว้ กลับไปใช้ AI ฟรี (หรือ key ใน .env)
func (h *Handler) DeleteKey(c *gin.Context) {
	if err := h.repo.SaveAI(c.Request.Context(), SavedAI{}); err != nil {
		helpers.ServerError(c, err)
		return
	}
	h.Status(c)
}

// Chat POST /api/v1/ai/chat — ส่งประวัติแชททั้งหมด ได้คำตอบ + งานที่ AI เสนอ
func (h *Handler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.WriteError(c, http.StatusBadRequest, "invalid_json")
		return
	}
	if code := req.Validate(); code != "" {
		helpers.WriteError(c, http.StatusBadRequest, code)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), chatTimeout)
	defer cancel()
	assistant, status, ok := h.ready(c, ctx)
	if !ok {
		return
	}
	tools := toolbox{shop: h.shop, repo: h.repo, today: h.now(), canSearch: status.WebSearch}
	reply, err := assistant.Chat(ctx, tools, req.Messages)
	if err != nil {
		writeAIError(c, err)
		return
	}
	c.JSON(http.StatusOK, reply)
}

// GetMarketCheck GET /api/v1/products/:id/market — ผลเช็คราคาตลาดครั้งล่าสุด (ไม่เสียเงิน)
func (h *Handler) GetMarketCheck(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "product_not_found")
	if !ok {
		return
	}
	check, err := h.repo.LatestMarketCheck(c.Request.Context(), id)
	if errors.Is(err, ErrNotFound) {
		helpers.WriteError(c, http.StatusNotFound, "market_check_not_found")
		return
	}
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, check)
}

// CheckMarket POST /api/v1/products/:id/market — ให้ AI ค้นราคาตลาดใหม่แล้วบันทึกผล (ต้องเป็น AI ที่ค้นเว็บได้)
func (h *Handler) CheckMarket(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "product_not_found")
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), marketTimeout)
	defer cancel()
	assistant, status, ok := h.ready(c, ctx)
	if !ok {
		return
	}
	if !status.WebSearch {
		writeAIError(c, ErrNoWebSearch)
		return
	}

	p, err := h.shop.Product(ctx, id)
	if errors.Is(err, product.ErrNotFound) {
		helpers.WriteError(c, http.StatusNotFound, "product_not_found")
		return
	}
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	prompt, err := h.researchPromptOf(ctx, p)
	if err != nil {
		helpers.ServerError(c, err)
		return
	}

	report, err := assistant.Research(ctx, prompt)
	if err != nil {
		writeAIError(c, err)
		return
	}
	check, err := h.repo.SaveMarketCheck(c.Request.Context(), p, report)
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusCreated, check)
}

// researchPromptOf รวมข้อมูลสินค้า + ใบต้นทุน + ช่องเพิ่มเติม + รูปแบบ VAT ของร้าน ไว้ส่งให้ AI
func (h *Handler) researchPromptOf(ctx context.Context, p product.Product) (string, error) {
	sheet, err := h.shop.CostSheet(ctx, p.ID)
	if err != nil {
		return "", err
	}
	fields, err := h.shop.Fields(ctx)
	if err != nil {
		return "", err
	}
	shop, err := h.shop.Company(ctx)
	if err != nil {
		return "", err
	}
	return researchPrompt(p, sheet, fields, shop), nil
}

// ready หา AI ที่ใช้ได้ ไม่มี → ตอบ 503 ai_not_configured
func (h *Handler) ready(c *gin.Context, ctx context.Context) (Assistant, Status, bool) {
	assistant, status, err := h.current(ctx)
	if err != nil {
		helpers.ServerError(c, err)
		return nil, status, false
	}
	if assistant == nil {
		helpers.WriteError(c, http.StatusServiceUnavailable, "ai_not_configured")
		return nil, status, false
	}
	return assistant, status, true
}
