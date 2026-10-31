package dashboard

import (
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

// RegisterRoutes ต่อ route ของหน้าภาพรวมเข้ากับกลุ่ม /api/v1 (ต้อง login)
func (h *Handler) RegisterRoutes(api gin.IRouter, authRequired gin.HandlerFunc) {
	api.GET("/dashboard", authRequired, h.Get)
}

// Get GET /api/v1/dashboard?month=2026-10 (ไม่ส่ง month = เดือนนี้)
func (h *Handler) Get(c *gin.Context) {
	monthStart, ok := helpers.MonthStart(c)
	if !ok {
		helpers.WriteError(c, http.StatusBadRequest, "month_invalid")
		return
	}

	dashboard, err := h.repo.Get(c.Request.Context(), monthStart)
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, dashboard)
}
