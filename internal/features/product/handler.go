package product

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

// RegisterRoutes ต่อ route ของสินค้าเข้ากับกลุ่ม /api/v1 (ทุก route ต้อง login)
func (h *Handler) RegisterRoutes(api gin.IRouter, authRequired gin.HandlerFunc) {
	api.GET("/products", authRequired, h.List)
	api.GET("/products/:id", authRequired, h.Get)
	api.POST("/products", authRequired, h.Create)
	api.PUT("/products/:id", authRequired, h.Update)
	api.DELETE("/products/:id", authRequired, h.Delete)
}

// List GET /api/v1/products
func (h *Handler) List(c *gin.Context) {
	products, err := h.repo.List(c.Request.Context())
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, products)
}

// Get GET /api/v1/products/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "product_not_found")
	if !ok {
		return
	}

	product, err := h.repo.FindByID(c.Request.Context(), id)
	if errors.Is(err, ErrNotFound) {
		helpers.WriteError(c, http.StatusNotFound, "product_not_found")
		return
	}
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, product)
}

// Create POST /api/v1/products
func (h *Handler) Create(c *gin.Context) {
	req, ok := readRequest(c)
	if !ok {
		return
	}

	product, err := h.repo.Create(c.Request.Context(), req)
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusCreated, product)
}

// Update PUT /api/v1/products/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "product_not_found")
	if !ok {
		return
	}
	req, ok := readRequest(c)
	if !ok {
		return
	}

	product, err := h.repo.Update(c.Request.Context(), id, req)
	if errors.Is(err, ErrNotFound) {
		helpers.WriteError(c, http.StatusNotFound, "product_not_found")
		return
	}
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, product)
}

// Delete DELETE /api/v1/products/:id
func (h *Handler) Delete(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "product_not_found")
	if !ok {
		return
	}

	err := h.repo.Delete(c.Request.Context(), id)
	if errors.Is(err, ErrNotFound) {
		helpers.WriteError(c, http.StatusNotFound, "product_not_found")
		return
	}
	if errors.Is(err, ErrInUse) {
		helpers.WriteError(c, http.StatusConflict, "product_in_use")
		return
	}
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// readRequest อ่าน JSON + Validate() ไม่ผ่าน → ตอบ 400 แล้วคืน ok = false
func readRequest(c *gin.Context) (ProductRequest, bool) {
	var req ProductRequest
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
