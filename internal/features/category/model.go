package category

import (
	"strings"
	"unicode/utf8"
)

// maxNameLength ชื่อหมวดยาวสุดกี่ตัวอักษร
const maxNameLength = 100

// Category หมวดหมู่สินค้า / product_count = มีสินค้ากี่ตัวในหมวดนี้ (ลบหมวด → สินค้าเหล่านี้กลายเป็นไม่มีหมวด)
type Category struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ProductCount int    `json:"product_count"`
}

// CategoryRequest ข้อมูลที่รับมาตอนเพิ่ม / แก้หมวด
type CategoryRequest struct {
	Name string `json:"name"`
}

// Validate เช็คข้อมูลก่อนบันทึก คืน error code ("" = ผ่าน)
func (req CategoryRequest) Validate() string {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return "category_name_required"
	}
	if utf8.RuneCountInString(name) > maxNameLength {
		return "category_name_too_long"
	}
	return ""
}

// cleanName ชื่อที่จะบันทึก (ตัดช่องว่างหน้า-หลัง)
func (req CategoryRequest) cleanName() string {
	return strings.TrimSpace(req.Name)
}
