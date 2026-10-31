package company

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/helpers"
)

// maxUploadBody ขนาด request ใหญ่สุดตอนอัปโหลดโลโก้ = โลโก้ 1 MB + เผื่อส่วนหัวของ form 64 KB
const maxUploadBody = maxLogoSize + 64<<10

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

// UploadLogo PUT /api/v1/company/logo — ส่งแบบ multipart/form-data ช่องชื่อ "logo"
func (h *Handler) UploadLogo(c *gin.Context) {
	data, code := readLogoFile(c)
	if code != "" {
		helpers.WriteError(c, http.StatusBadRequest, code)
		return
	}

	contentType := logoContentType(data)
	if contentType == "" {
		helpers.WriteError(c, http.StatusBadRequest, "logo_type_invalid")
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

// readLogoFile อ่านไฟล์จากช่อง "logo" ของ form
// คืน error code ถ้าไม่มีไฟล์ ("logo_required") หรือไฟล์ใหญ่เกิน ("logo_too_large")
func readLogoFile(c *gin.Context) ([]byte, string) {
	// กันไฟล์ใหญ่เกินตั้งแต่ตอนรับ ไม่ต้องรอรับไฟล์ทั้งก้อนก่อนค่อยเช็ค
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBody)

	fileHeader, err := c.FormFile("logo")
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return nil, "logo_too_large"
	}
	if err != nil {
		return nil, "logo_required"
	}
	if fileHeader.Size > maxLogoSize {
		return nil, "logo_too_large"
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, "logo_required"
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil || len(data) == 0 {
		return nil, "logo_required"
	}
	return data, ""
}
