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
	api.GET("/cost-presets", authRequired, h.ListPresets)
	api.POST("/cost-presets", authRequired, h.CreatePreset)
	api.PUT("/cost-presets/:id", authRequired, h.UpdatePreset)
	api.DELETE("/cost-presets/:id", authRequired, h.DeletePreset)
}

// Get GET /api/v1/products/:id/costs
func (h *Handler) Get(c *gin.Context) {
	productID, ok := helpers.UUIDParam(c, "id", "product_not_found")
	if !ok {
		return
	}

	sheet, err := h.repo.Get(c.Request.Context(), productID)
	if writeRepoError(c, err) {
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

	req.Items, err = h.repo.ResolveMaterials(c.Request.Context(), productID, req.Items)
	if writeRepoError(c, err) {
		return
	}
	amounts := itemAmounts(req.Items)
	sheet, err := h.repo.Save(c.Request.Context(), productID, req, amounts, costPrice(req.Items, amounts, req.BatchQuantity))
	if writeRepoError(c, err) {
		return
	}
	c.JSON(http.StatusOK, sheet)
}

// ListPresets GET /api/v1/cost-presets
func (h *Handler) ListPresets(c *gin.Context) {
	presets, err := h.repo.ListPresets(c.Request.Context())
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, presets)
}

// CreatePreset POST /api/v1/cost-presets
func (h *Handler) CreatePreset(c *gin.Context) {
	req, ok := readPreset(c)
	if !ok {
		return
	}

	preset, err := h.repo.CreatePreset(c.Request.Context(), req)
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusCreated, preset)
}

// UpdatePreset PUT /api/v1/cost-presets/:id
func (h *Handler) UpdatePreset(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "cost_preset_not_found")
	if !ok {
		return
	}
	req, ok := readPreset(c)
	if !ok {
		return
	}

	preset, err := h.repo.UpdatePreset(c.Request.Context(), id, req)
	if writeRepoError(c, err) {
		return
	}
	c.JSON(http.StatusOK, preset)
}

// DeletePreset DELETE /api/v1/cost-presets/:id
func (h *Handler) DeletePreset(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "cost_preset_not_found")
	if !ok {
		return
	}

	err := h.repo.DeletePreset(c.Request.Context(), id)
	if writeRepoError(c, err) {
		return
	}
	c.Status(http.StatusNoContent)
}

// readPreset อ่าน JSON + Validate() ไม่ผ่าน → ตอบ 400 แล้วคืน ok = false
func readPreset(c *gin.Context) (PresetRequest, bool) {
	var req PresetRequest
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

// writeRepoError ตอบ error จาก repository คืน true ถ้ามี error (handler ต้อง return)
func writeRepoError(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, ErrProductNotFound):
		helpers.WriteError(c, http.StatusNotFound, "product_not_found")
	case errors.Is(err, ErrPresetNotFound):
		helpers.WriteError(c, http.StatusNotFound, "cost_preset_not_found")
	case errors.Is(err, ErrMaterialInvalid):
		helpers.WriteError(c, http.StatusBadRequest, "cost_material_invalid")
	default:
		helpers.ServerError(c, err)
	}
	return true
}
