package cost

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

// RegisterRoutes ต่อ route ของหน้าคำนวณต้นทุนเข้ากับกลุ่ม /api/v1 (ทุก route ต้อง login)
func (h *Handler) RegisterRoutes(api gin.IRouter, authRequired gin.HandlerFunc) {
	api.GET("/products/:id/costs", authRequired, h.Get)
	api.PUT("/products/:id/costs", authRequired, h.Save)
}

// Get GET /api/v1/products/:id/costs
func (h *Handler) Get(c *gin.Context) {
	productID, ok := helpers.UUIDParam(c, "id", "product_not_found")
	if !ok {
		return
	}

	sheet, err := h.repo.Get(c.Request.Context(), productID)
	if errors.Is(err, ErrProductNotFound) {
		helpers.WriteError(c, http.StatusNotFound, "product_not_found")
		return
	}
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, sheet)
}

// Save PUT /api/v1/products/:id/costs — แทนที่รายการต้นทุนทั้งหมด แล้วคำนวณต้นทุนต่อชิ้นใหม่
func (h *Handler) Save(c *gin.Context) {
	productID, ok := helpers.UUIDParam(c, "id", "product_not_found")
	if !ok {
		return
	}

	var req CostSheetRequest
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

	sheet, err := h.repo.Save(c.Request.Context(), productID, req, costPrice(req))
	if errors.Is(err, ErrProductNotFound) {
		helpers.WriteError(c, http.StatusNotFound, "product_not_found")
		return
	}
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, sheet)
}
