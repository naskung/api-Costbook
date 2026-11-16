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

// Get GET /api/v1/dashboard?from=2026-11-01&to=2026-11-16 (ไม่ส่ง from / to = ทั้งเดือนนี้)
func (h *Handler) Get(c *gin.Context) {
	period, ok := helpers.DateRangeParams(c)
	if !ok {
		return
	}

	dashboard, err := h.repo.Get(c.Request.Context(), period)
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, dashboard)
}
