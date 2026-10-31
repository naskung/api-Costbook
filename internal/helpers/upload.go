package helpers

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/utils"
)

// MaxImageSize ขนาดรูปใหญ่สุด 1 MB (โลโก้ร้าน / รูปสินค้า)
const MaxImageSize = 1 << 20

// maxUploadBody ขนาด request ใหญ่สุดตอนอัปโหลดรูป = รูป 1 MB + เผื่อส่วนหัวของ form 64 KB
const maxUploadBody = MaxImageSize + 64<<10

// ReadImage อ่านรูปจากช่อง field ของ multipart/form-data
// คืน (ไฟล์, ชนิดรูป, ปัญหา) — ปัญหา: "required" = ไม่มีไฟล์, "too_large" = เกิน 1 MB,
// "type_invalid" = ไม่ใช่ PNG / JPG / WebP, "" = ผ่าน
// คนเรียกเอาปัญหาไปต่อท้ายชื่อ error เอง เช่น "logo_" + ปัญหา
func ReadImage(c *gin.Context, field string) ([]byte, string, string) {
	// กันไฟล์ใหญ่เกินตั้งแต่ตอนรับ ไม่ต้องรอรับไฟล์ทั้งก้อนก่อนค่อยเช็ค
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBody)

	fileHeader, err := c.FormFile(field)
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return nil, "", "too_large"
	}
	if err != nil {
		return nil, "", "required"
	}
	if fileHeader.Size > MaxImageSize {
		return nil, "", "too_large"
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, "", "required"
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil || len(data) == 0 {
		return nil, "", "required"
	}

	contentType := utils.ImageContentType(data)
	if contentType == "" {
		return nil, "", "type_invalid"
	}
	return data, contentType, ""
}
