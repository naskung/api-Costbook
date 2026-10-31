package ai

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/features/product"
	"github.com/naskung/api-Costbook/internal/helpers"
)

// เวลารอสูงสุดของงาน AI (เช็คราคาตลาดค้นเว็บหลายรอบ ใช้นานกว่าแชท)
const (
	marketTimeout = 5 * time.Minute
	chatTimeout   = 3 * time.Minute
)

type Handler struct {
	repo   Repository
	claude Claude // nil = ยังไม่ได้ใส่ ANTHROPIC_API_KEY
	shop   ShopReader
	now    func() time.Time
}

func NewHandler(repo Repository, claude Claude, shop ShopReader) *Handler {
	return &Handler{repo: repo, claude: claude, shop: shop, now: time.Now}
}

// RegisterRoutes ต่อ route ของ AI เข้ากับกลุ่ม /api/v1 (ต้อง login ทุกเส้น)
func (h *Handler) RegisterRoutes(api gin.IRouter, authRequired gin.HandlerFunc) {
	api.GET("/ai/status", authRequired, h.Status)
	api.POST("/ai/chat", authRequired, h.Chat)
	api.GET("/products/:id/market", authRequired, h.GetMarketCheck)
	api.POST("/products/:id/market", authRequired, h.CheckMarket)
}

// Status GET /api/v1/ai/status — เปิดใช้ AI ได้ไหม (Flutter ใช้ซ่อน / ปิดปุ่ม AI)
func (h *Handler) Status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"enabled": h.claude != nil, "model": model})
}

// Chat POST /api/v1/ai/chat — ส่งประวัติแชททั้งหมด ได้คำตอบ + งานที่ AI เสนอ
func (h *Handler) Chat(c *gin.Context) {
	if !h.aiReady(c) {
		return
	}
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
	tools := toolbox{shop: h.shop, repo: h.repo, today: h.now()}
	reply, err := chat(ctx, h.claude, tools, req.Messages)
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

// CheckMarket POST /api/v1/products/:id/market — ให้ AI ค้นราคาตลาดใหม่แล้วบันทึกผล (มีค่าใช้จ่าย ใช้เวลา 1–3 นาที)
func (h *Handler) CheckMarket(c *gin.Context) {
	if !h.aiReady(c) {
		return
	}
	id, ok := helpers.UUIDParam(c, "id", "product_not_found")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), marketTimeout)
	defer cancel()
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

	report, err := research(ctx, h.claude, prompt)
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

// aiReady ยังไม่ได้ใส่ API key → ตอบ 503 ai_not_configured
func (h *Handler) aiReady(c *gin.Context) bool {
	if h.claude == nil {
		helpers.WriteError(c, http.StatusServiceUnavailable, "ai_not_configured")
		return false
	}
	return true
}
