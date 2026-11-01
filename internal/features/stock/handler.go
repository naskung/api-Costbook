package stock

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/helpers"
	"github.com/naskung/api-Costbook/internal/utils"
)

type Handler struct {
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// RegisterRoutes ต่อ route ของสต็อกเข้ากับกลุ่ม /api/v1 (ทุก route ต้อง login)
func (h *Handler) RegisterRoutes(api gin.IRouter, authRequired gin.HandlerFunc) {
	api.GET("/stock", authRequired, h.List)
	api.GET("/stock/movements", authRequired, h.Movements)
	api.POST("/stock/in", authRequired, h.StockIn)
	api.POST("/stock/out", authRequired, h.StockOut)
}

// List GET /api/v1/stock?low=true (low=true = เฉพาะใกล้หมด)
func (h *Handler) List(c *gin.Context) {
	onlyLow := c.Query("low") == "true"

	items, err := h.repo.List(c.Request.Context(), onlyLow)
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

// Movements GET /api/v1/stock/movements?product_id=&limit=20&offset=0 (ไม่ส่ง product_id = ทุกสินค้า / ไม่ส่ง limit = ล่าสุด 200)
func (h *Handler) Movements(c *gin.Context) {
	productID := c.Query("product_id")
	if productID != "" && !utils.IsUUID(productID) {
		helpers.WriteError(c, http.StatusNotFound, "product_not_found")
		return
	}
	limit, offset, ok := helpers.PageParams(c)
	if !ok {
		return
	}

	movements, err := h.repo.Movements(c.Request.Context(), productID, limit, offset)
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, movements)
}

// StockIn POST /api/v1/stock/in
func (h *Handler) StockIn(c *gin.Context) {
	h.addMovement(c, "in")
}

// StockOut POST /api/v1/stock/out
func (h *Handler) StockOut(c *gin.Context) {
	h.addMovement(c, "out")
}

// addMovement ใช้ร่วมกันทั้งรับเข้าและเบิกออก ต่างกันแค่ movementType
func (h *Handler) addMovement(c *gin.Context, movementType string) {
	var req MovementRequest
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

	movement, err := h.repo.AddMovement(c.Request.Context(), movementType, req)
	if errors.Is(err, ErrProductNotFound) {
		helpers.WriteError(c, http.StatusNotFound, "product_not_found")
		return
	}
	if errors.Is(err, ErrNotEnough) {
		helpers.WriteError(c, http.StatusConflict, "stock_not_enough")
		return
	}
	if errors.Is(err, ErrNotTracked) {
		helpers.WriteError(c, http.StatusConflict, "stock_not_tracked")
		return
	}
	if WriteShortage(c, err) {
		return
	}
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusCreated, movement)
}
