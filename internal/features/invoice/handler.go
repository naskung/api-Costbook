package invoice

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/features/stock"
	"github.com/naskung/api-Costbook/internal/helpers"
)

type Handler struct {
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// RegisterRoutes ต่อ route ของเอกสารขาย (ใบกำกับภาษี / ใบเสร็จรับเงิน) เข้ากับกลุ่ม /api/v1 (ทุก route ต้อง login)
// ไม่มี PUT / DELETE เพราะเอกสารขายแก้ / ลบไม่ได้ ผิดให้ยกเลิกแล้วออกใบใหม่
func (h *Handler) RegisterRoutes(api gin.IRouter, authRequired gin.HandlerFunc) {
	api.GET("/invoices", authRequired, h.List)
	api.GET("/invoices/:id", authRequired, h.Get)
	api.POST("/invoices", authRequired, h.Create)
	api.POST("/invoices/:id/cancel", authRequired, h.Cancel)
}

// List GET /api/v1/invoices?month=2026-10 (ไม่ส่ง month = เดือนนี้)
func (h *Handler) List(c *gin.Context) {
	monthStart, ok := helpers.MonthStart(c)
	if !ok {
		helpers.WriteError(c, http.StatusBadRequest, "month_invalid")
		return
	}

	invoices, err := h.repo.List(c.Request.Context(), monthStart)
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, invoices)
}

// Get GET /api/v1/invoices/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "invoice_not_found")
	if !ok {
		return
	}

	invoice, err := h.repo.FindByID(c.Request.Context(), id)
	if writeRepoError(c, err) {
		return
	}
	c.JSON(http.StatusOK, invoice)
}

// Create POST /api/v1/invoices
func (h *Handler) Create(c *gin.Context) {
	var req InvoiceRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		helpers.WriteError(c, http.StatusBadRequest, "invalid_json")
		return
	}
	code := req.Validate()
	if code != "" {
		helpers.WriteError(c, http.StatusBadRequest, code)
		return
	}

	invoice, err := h.repo.Create(c.Request.Context(), req)
	if writeRepoError(c, err) {
		return
	}
	c.JSON(http.StatusCreated, invoice)
}

// Cancel POST /api/v1/invoices/:id/cancel
func (h *Handler) Cancel(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "invoice_not_found")
	if !ok {
		return
	}

	invoice, err := h.repo.Cancel(c.Request.Context(), id)
	if writeRepoError(c, err) {
		return
	}
	c.JSON(http.StatusOK, invoice)
}

// writeRepoError ตอบ error จาก repository คืน true ถ้ามี error (handler ต้อง return)
func writeRepoError(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, ErrNotFound):
		helpers.WriteError(c, http.StatusNotFound, "invoice_not_found")
	case errors.Is(err, ErrProductNotFound):
		helpers.WriteError(c, http.StatusNotFound, "product_not_found")
	case errors.Is(err, ErrStockNotEnough):
		helpers.WriteError(c, http.StatusConflict, "stock_not_enough")
	case errors.Is(err, ErrNotForSale):
		helpers.WriteError(c, http.StatusConflict, "product_not_for_sale")
	case stock.WriteShortage(c, err):
		// วัสดุไม่พอทำงานที่ไม่นับสต็อก → ตอบไปแล้วใน WriteShortage
	case errors.Is(err, ErrAlreadyCancelled):
		helpers.WriteError(c, http.StatusConflict, "invoice_already_cancelled")
	case errors.Is(err, ErrSellerTaxIDMissing):
		helpers.WriteError(c, http.StatusConflict, "company_tax_id_required")
	case errors.Is(err, ErrCustomerRequired):
		helpers.WriteError(c, http.StatusBadRequest, "invoice_customer_required")
	default:
		helpers.ServerError(c, err)
	}
	return true
}
