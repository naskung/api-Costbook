package ai

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
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
}

func NewHandler(repo Repository, shop ShopReader, cfg Config) *Handler {
	return &Handler{repo: repo, shop: shop, cfg: cfg, now: time.Now, newClaude: NewClaude, newGemini: NewGemini}
}

// RegisterRoutes ต่อ route ของ AI เข้ากับกลุ่ม /api/v1 (ต้อง login ทุกเส้น)
func (h *Handler) RegisterRoutes(api gin.IRouter, authRequired gin.HandlerFunc) {
	api.GET("/ai/status", authRequired, h.Status)
	api.PUT("/ai/key", authRequired, h.SaveKey)
	api.DELETE("/ai/key", authRequired, h.DeleteKey)
	api.POST("/ai/chat", authRequired, h.Chat)
	api.GET("/products/:id/market", authRequired, h.GetMarketCheck)
	api.POST("/products/:id/market", authRequired, h.CheckMarket)
}

// current AI ที่ใช้อยู่ตอนนี้ + สถานะ (assistant = nil เมื่อยังไม่มี AI ให้ใช้)
func (h *Handler) current(ctx context.Context) (Assistant, Status, error) {
	savedKey, err := h.savedKey(ctx)
	if err != nil {
		return nil, Status{}, err
	}
	status := pickProvider(savedKey, h.cfg)
	switch {
	case status.Provider == providerClaude && status.Source == sourceSettings:
		return claudeAssistant{claude: h.newClaude(savedKey)}, status, nil
	case status.Provider == providerClaude:
		return claudeAssistant{claude: h.newClaude(h.cfg.ClaudeKey)}, status, nil
	case status.Provider == providerGemini:
		return geminiAssistant{gemini: h.newGemini(h.cfg.GeminiKey, status.Model)}, status, nil
	}
	return nil, status, nil
}

// savedKey key ที่ร้านใส่ไว้ (ถอดรหัสแล้ว) — ถอดไม่ได้ (เช่น เปลี่ยน JWT_SECRET) ถือว่าไม่มี ให้ร้านใส่ใหม่
func (h *Handler) savedKey(ctx context.Context) (string, error) {
	encrypted, err := h.repo.SavedClaudeKey(ctx)
	if err != nil || encrypted == "" {
		return "", err
	}
	key, err := utils.Decrypt(encrypted, h.cfg.Secret)
	if err != nil {
		slog.Warn("ถอดรหัส key ของ AI ไม่ได้ (เปลี่ยน JWT_SECRET?) ให้ใส่ key ใหม่ในหน้าตั้งค่า")
		return "", nil
	}
	return key, nil
}

// Status GET /api/v1/ai/status — AI ที่ใช้อยู่ (Claude / Gemini ฟรี / ไม่มี) ค้นเว็บได้ไหม ใส่ key เองไว้ไหม
func (h *Handler) Status(c *gin.Context) {
	_, status, err := h.current(c.Request.Context())
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, status)
}

// SaveKey PUT /api/v1/ai/key {"api_key": "sk-ant-..."} — ลอง key กับ Claude ก่อน ใช้ได้จริงถึงเก็บ (เข้ารหัส)
func (h *Handler) SaveKey(c *gin.Context) {
	var req struct {
		APIKey string `json:"api_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.WriteError(c, http.StatusBadRequest, "invalid_json")
		return
	}
	key := strings.TrimSpace(req.APIKey)
	if !validClaudeKey(key) {
		helpers.WriteError(c, http.StatusBadRequest, "ai_key_format_invalid")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), keyCheckTimeout)
	defer cancel()
	err := h.newClaude(key).CheckKey(ctx)
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) && (apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden) {
		helpers.WriteError(c, http.StatusBadRequest, "ai_key_invalid")
		return
	}
	if err != nil {
		writeAIError(c, err)
		return
	}

	encrypted, err := utils.Encrypt(key, h.cfg.Secret)
	if err == nil {
		err = h.repo.SaveClaudeKey(c.Request.Context(), encrypted)
	}
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	h.Status(c)
}

// DeleteKey DELETE /api/v1/ai/key — ลบ key ที่ร้านใส่ไว้ กลับไปใช้ AI ฟรี (หรือ key ใน .env)
func (h *Handler) DeleteKey(c *gin.Context) {
	if err := h.repo.SaveClaudeKey(c.Request.Context(), ""); err != nil {
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
