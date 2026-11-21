package company

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

// RegisterRoutes ต่อ route ของข้อมูลร้านเข้ากับกลุ่ม /api/v1
// ดูข้อมูลร้าน / โลโก้ ไม่ต้อง login → หน้า login แสดงชื่อและโลโก้ร้านได้
// แก้ไขต้อง login (authRequired)
func (h *Handler) RegisterRoutes(api gin.IRouter, authRequired gin.HandlerFunc) {
	api.GET("/company", h.Get)
	api.PUT("/company", authRequired, h.Update)
	api.GET("/company/logo", h.Logo)
	api.GET("/company/icon", h.Icon)
	api.PUT("/company/logo", authRequired, h.UploadLogo)
	api.DELETE("/company/logo", authRequired, h.DeleteLogo)
}

// Get GET /api/v1/company
func (h *Handler) Get(c *gin.Context) {
	company, err := h.repo.Get(c.Request.Context())
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, company)
}

// Update PUT /api/v1/company
func (h *Handler) Update(c *gin.Context) {
	var req CompanyRequest
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

	company, err := h.repo.Update(c.Request.Context(), req)
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, company)
}

// Logo GET /api/v1/company/logo — ตอบเป็นรูปตรง ๆ (ใช้กับ Image.network ได้เลย)
func (h *Handler) Logo(c *gin.Context) {
	logo, err := h.repo.FindLogo(c.Request.Context())
	if errors.Is(err, ErrLogoNotFound) {
		helpers.WriteError(c, http.StatusNotFound, "logo_not_found")
		return
	}
	if err != nil {
		helpers.ServerError(c, err)
		return
	}

	c.Header("Cache-Control", "no-cache")         // เปลี่ยนโลโก้แล้ว browser ต้องถามใหม่ ไม่ใช้รูปเก่า
	c.Header("X-Content-Type-Options", "nosniff") // บอก browser ว่าเป็นรูปเท่านั้น ห้ามเดาเป็นชนิดอื่น
	c.Data(http.StatusOK, logo.ContentType, logo.Data)
}

// Icon GET /api/v1/company/icon?size=192 — ไอคอนแอป / favicon PNG สี่เหลี่ยมจัตุรัส (ไม่ส่ง size = 192)
// ไม่ต้อง login เพราะ browser / มือถือโหลดไอคอนเองโดยไม่ส่ง token
// มีโลโก้ร้าน = โลโก้วางกลางพื้นขาว / ยังไม่มี = ไอคอนเริ่มต้นของ CostBook (ตอบ 200 เสมอ ไม่ตอบ 404)
func (h *Handler) Icon(c *gin.Context) {
	size, ok := iconSize(c.DefaultQuery("size", "192"))
	if !ok {
		helpers.WriteError(c, http.StatusBadRequest, "icon_size_invalid")
		return
	}
	logo, err := h.repo.FindLogo(c.Request.Context())
	if err != nil && !errors.Is(err, ErrLogoNotFound) {
		helpers.ServerError(c, err)
		return
	}

	icon, err := iconPNG(logo.Data, size)
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.Header("Cache-Control", "no-cache") // เปลี่ยนโลโก้แล้ว browser ต้องถามใหม่
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "image/png", icon)
}

// UploadLogo PUT /api/v1/company/logo — ส่งแบบ multipart/form-data ช่องชื่อ "logo"
func (h *Handler) UploadLogo(c *gin.Context) {
	data, contentType, problem := helpers.ReadImage(c, "logo")
	if problem != "" {
		helpers.WriteError(c, http.StatusBadRequest, "logo_"+problem)
		return
	}

	company, err := h.repo.SaveLogo(c.Request.Context(), Logo{Data: data, ContentType: contentType})
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, company)
}

// DeleteLogo DELETE /api/v1/company/logo
func (h *Handler) DeleteLogo(c *gin.Context) {
	err := h.repo.DeleteLogo(c.Request.Context())
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
