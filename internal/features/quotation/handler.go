package quotation

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/helpers"
)

type Handler struct {
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// RegisterRoutes ต่อ route ของใบเสนอราคาเข้ากับกลุ่ม /api/v1 (ทุก route ต้อง login)
// ใบเสนอราคาแก้ / ลบไม่ได้ ผิดให้ยกเลิกแล้วออกใบใหม่
func (h *Handler) RegisterRoutes(api gin.IRouter, authRequired gin.HandlerFunc) {
	api.GET("/quotations", authRequired, h.List)
	api.GET("/quotations/:id", authRequired, h.Get)
	api.POST("/quotations", authRequired, h.Create)
	api.POST("/quotations/:id/cancel", authRequired, h.Cancel)
}

// List GET /api/v1/quotations?from=2026-11-01&to=2026-11-21&limit=20&offset=0
// ไม่ส่ง from / to = ทั้งเดือนนี้ / ไม่ส่ง limit = ทั้งช่วง
func (h *Handler) List(c *gin.Context) {
	period, ok := helpers.DateRangeParams(c)
	if !ok {
		return
	}
	limit, offset, ok := helpers.PageParams(c)
	if !ok {
		return
	}

	quotations, err := h.repo.List(c.Request.Context(), period, limit, offset)
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, quotations)
}

// Get GET /api/v1/quotations/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "quotation_not_found")
	if !ok {
		return
	}

	quotation, err := h.repo.FindByID(c.Request.Context(), id)
	if writeRepoError(c, err) {
		return
	}
	c.JSON(http.StatusOK, quotation)
}

// Create POST /api/v1/quotations
func (h *Handler) Create(c *gin.Context) {
	var req QuotationRequest
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

	quotation, err := h.repo.Create(c.Request.Context(), req)
	if writeRepoError(c, err) {
		return
	}
	c.JSON(http.StatusCreated, quotation)
}

// Cancel POST /api/v1/quotations/:id/cancel
func (h *Handler) Cancel(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "quotation_not_found")
	if !ok {
		return
	}

	quotation, err := h.repo.Cancel(c.Request.Context(), id)
	if writeRepoError(c, err) {
		return
	}
	c.JSON(http.StatusOK, quotation)
}

// writeRepoError ตอบ error จาก repository คืน true ถ้ามี error (handler ต้อง return)
func writeRepoError(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, ErrNotFound):
		helpers.WriteError(c, http.StatusNotFound, "quotation_not_found")
	case errors.Is(err, ErrProductNotFound):
		helpers.WriteError(c, http.StatusNotFound, "product_not_found")
	case errors.Is(err, ErrNotForSale):
		helpers.WriteError(c, http.StatusConflict, "product_not_for_sale")
	case errors.Is(err, ErrAlreadyCancelled):
		helpers.WriteError(c, http.StatusConflict, "quotation_already_cancelled")
	default:
		helpers.ServerError(c, err)
	}
	return true
}
