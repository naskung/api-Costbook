package field

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

// RegisterRoutes ต่อ route ของช่องข้อมูลเพิ่มเติมเข้ากับกลุ่ม /api/v1 (ทุก route ต้อง login)
func (h *Handler) RegisterRoutes(api gin.IRouter, authRequired gin.HandlerFunc) {
	api.GET("/product-fields", authRequired, h.List)
	api.POST("/product-fields", authRequired, h.Create)
	api.PUT("/product-fields/:id", authRequired, h.Update)
	api.DELETE("/product-fields/:id", authRequired, h.Delete)
	api.GET("/product-form-layout", authRequired, h.Layout)
	api.PUT("/product-form-layout", authRequired, h.SaveLayout)
}

// List GET /api/v1/product-fields
func (h *Handler) List(c *gin.Context) {
	fields, err := h.repo.List(c.Request.Context())
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, fields)
}

// Create POST /api/v1/product-fields
func (h *Handler) Create(c *gin.Context) {
	req, ok := readRequest(c)
	if !ok {
		return
	}

	field, err := h.repo.Create(c.Request.Context(), req)
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusCreated, field)
}

// Update PUT /api/v1/product-fields/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "field_not_found")
	if !ok {
		return
	}
	req, ok := readRequest(c)
	if !ok {
		return
	}

	field, err := h.repo.Update(c.Request.Context(), id, req)
	if writeRepoError(c, err) {
		return
	}
	c.JSON(http.StatusOK, field)
}

// Delete DELETE /api/v1/product-fields/:id — ค่าของช่องนี้ในสินค้าทุกตัวถูกลบด้วย
func (h *Handler) Delete(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "field_not_found")
	if !ok {
		return
	}

	err := h.repo.Delete(c.Request.Context(), id)
	if writeRepoError(c, err) {
		return
	}
	c.Status(http.StatusNoContent)
}

// Layout GET /api/v1/product-form-layout — ลำดับทุกช่องในฟอร์มสินค้า [{"key": "name", "sort_order": 2}, {"key": "<id ช่อง>", ...}]
func (h *Handler) Layout(c *gin.Context) {
	items, err := h.repo.Layout(c.Request.Context())
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

// SaveLayout PUT /api/v1/product-form-layout {"items": [{"key", "sort_order"}, ...]} — ส่งครบทุกช่อง ได้ลำดับใหม่กลับไป
func (h *Handler) SaveLayout(c *gin.Context) {
	var req LayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.WriteError(c, http.StatusBadRequest, "invalid_json")
		return
	}
	if code := req.Validate(); code != "" {
		helpers.WriteError(c, http.StatusBadRequest, code)
		return
	}
	err := h.repo.SaveLayout(c.Request.Context(), req.Items)
	if errors.Is(err, ErrLayoutInvalid) {
		helpers.WriteError(c, http.StatusBadRequest, "form_layout_invalid")
		return
	}
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	h.Layout(c)
}

// readRequest อ่าน JSON + Validate() ไม่ผ่าน → ตอบ 400 แล้วคืน ok = false
func readRequest(c *gin.Context) (FieldRequest, bool) {
	var req FieldRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		helpers.WriteError(c, http.StatusBadRequest, "invalid_json")
		return req, false
	}

	code := req.Validate()
	if code != "" {
		helpers.WriteError(c, http.StatusBadRequest, code)
		return req, false
	}
	return req, true
}

// writeRepoError ตอบ error จาก Update / Delete คืน true ถ้ามี error (handler ต้อง return)
func writeRepoError(c *gin.Context, err error) bool {
	if errors.Is(err, ErrNotFound) {
		helpers.WriteError(c, http.StatusNotFound, "field_not_found")
		return true
	}
	if err != nil {
		helpers.ServerError(c, err)
		return true
	}
	return false
}
