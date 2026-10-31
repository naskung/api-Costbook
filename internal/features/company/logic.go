package company

import "net/http"

// maxLogoSize ขนาดโลโก้ใหญ่สุด 1 MB
const maxLogoSize = 1 << 20

// logoContentType ดูจากข้างในไฟล์ (ไม่เชื่อนามสกุลไฟล์) ว่าเป็นรูปชนิดไหน
// รับแค่ PNG / JPG / WebP ถ้าเป็นอย่างอื่นคืน ""
func logoContentType(data []byte) string {
	contentType := http.DetectContentType(data)
	switch contentType {
	case "image/png", "image/jpeg", "image/webp":
		return contentType
	}
	return ""
}
