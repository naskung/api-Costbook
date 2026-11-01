package transaction

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

// RegisterRoutes ต่อ route ของรายรับ-รายจ่ายเข้ากับกลุ่ม /api/v1 (ทุก route ต้อง login)
func (h *Handler) RegisterRoutes(api gin.IRouter, authRequired gin.HandlerFunc) {
	api.GET("/transactions", authRequired, h.List)
	api.GET("/transactions/summary", authRequired, h.Summary)
	api.POST("/transactions", authRequired, h.Create)
	api.PUT("/transactions/:id", authRequired, h.Update)
	api.DELETE("/transactions/:id", authRequired, h.Delete)
}

// List GET /api/v1/transactions?month=2026-10 (ไม่ส่ง month = เดือนนี้)
func (h *Handler) List(c *gin.Context) {
	monthStart, ok := helpers.MonthStart(c)
	if !ok {
		helpers.WriteError(c, http.StatusBadRequest, "month_invalid")
		return
	}

	transactions, err := h.repo.List(c.Request.Context(), monthStart)
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, transactions)
}

// Summary GET /api/v1/transactions/summary?month=2026-10
func (h *Handler) Summary(c *gin.Context) {
	monthStart, ok := helpers.MonthStart(c)
	if !ok {
		helpers.WriteError(c, http.StatusBadRequest, "month_invalid")
		return
	}

	amounts, err := h.repo.MonthAmounts(c.Request.Context(), monthStart)
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, NewSummary(monthStart[:7], amounts)) // "2026-10-01" → "2026-10"
}

// Create POST /api/v1/transactions
func (h *Handler) Create(c *gin.Context) {
	req, ok := readRequest(c)
	if !ok {
		return
	}

	transaction, err := h.repo.Create(c.Request.Context(), req)
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusCreated, transaction)
}

// Update PUT /api/v1/transactions/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "transaction_not_found")
	if !ok {
		return
	}
	req, ok := readRequest(c)
	if !ok {
		return
	}

	transaction, err := h.repo.Update(c.Request.Context(), id, req)
	if writeRepoError(c, err) {
		return
	}
	c.JSON(http.StatusOK, transaction)
}

// Delete DELETE /api/v1/transactions/:id
func (h *Handler) Delete(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "transaction_not_found")
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
func readRequest(c *gin.Context) (TransactionRequest, bool) {
	var req TransactionRequest
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
		helpers.WriteError(c, http.StatusNotFound, "transaction_not_found")
		return true
	}
	if errors.Is(err, ErrFromInvoice) {
		helpers.WriteError(c, http.StatusConflict, "transaction_from_invoice")
		return true
	}
	if err != nil {
		helpers.ServerError(c, err)
		return true
	}
	return false
}
