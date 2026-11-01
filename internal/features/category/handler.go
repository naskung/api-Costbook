package category

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

// RegisterRoutes ต่อ route ของหมวดหมู่สินค้าเข้ากับกลุ่ม /api/v1 (ทุก route ต้อง login)
func (h *Handler) RegisterRoutes(api gin.IRouter, authRequired gin.HandlerFunc) {
	api.GET("/product-categories", authRequired, h.List)
	api.POST("/product-categories", authRequired, h.Create)
	api.PUT("/product-categories/:id", authRequired, h.Update)
	api.DELETE("/product-categories/:id", authRequired, h.Delete)
}

// List GET /api/v1/product-categories?limit=20&offset=0 (ไม่ส่ง limit = ทั้งหมด เช่น ปุ่มกรอง / ช่องเลือกหมวด)
func (h *Handler) List(c *gin.Context) {
	limit, offset, ok := helpers.PageParams(c)
	if !ok {
		return
	}
	categories, err := h.repo.List(c.Request.Context(), limit, offset)
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, categories)
}

// Create POST /api/v1/product-categories
func (h *Handler) Create(c *gin.Context) {
	req, ok := readRequest(c)
	if !ok {
		return
	}

	category, err := h.repo.Create(c.Request.Context(), req)
	if writeRepoError(c, err) {
		return
	}
	c.JSON(http.StatusCreated, category)
}

// Update PUT /api/v1/product-categories/:id (เปลี่ยนชื่อหมวด)
func (h *Handler) Update(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "category_not_found")
	if !ok {
		return
	}
	req, ok := readRequest(c)
	if !ok {
		return
	}

	category, err := h.repo.Update(c.Request.Context(), id, req)
	if writeRepoError(c, err) {
		return
	}
	c.JSON(http.StatusOK, category)
}

// Delete DELETE /api/v1/product-categories/:id — สินค้าในหมวดนี้กลายเป็นไม่มีหมวด
func (h *Handler) Delete(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "category_not_found")
	if !ok {
		return
	}

	err := h.repo.Delete(c.Request.Context(), id)
	if writeRepoError(c, err) {
		return
	}
	c.Status(http.StatusNoContent)
}

// readRequest อ่าน JSON + Validate() ไม่ผ่าน → ตอบ 400 แล้วคืน ok = false
func readRequest(c *gin.Context) (CategoryRequest, bool) {
	var req CategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.WriteError(c, http.StatusBadRequest, "invalid_json")
		return req, false
	}
	if code := req.Validate(); code != "" {
		helpers.WriteError(c, http.StatusBadRequest, code)
		return req, false
	}
	return req, true
}

// writeRepoError ตอบ error จาก repository คืน true ถ้ามี error (handler ต้อง return)
func writeRepoError(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, ErrNotFound):
		helpers.WriteError(c, http.StatusNotFound, "category_not_found")
	case errors.Is(err, ErrNameTaken):
		helpers.WriteError(c, http.StatusConflict, "category_name_taken")
	default:
		helpers.ServerError(c, err)
	}
	return true
}
