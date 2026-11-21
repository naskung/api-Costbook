package barcode

import "strings"

// isBarcode เช็คว่าเป็นเลขบาร์โค้ดสากล: ตัวเลขล้วน 8 หลัก (EAN-8), 12 หลัก (UPC-A), 13 หลัก (EAN-13), 14 หลัก (GTIN-14)
func isBarcode(code string) bool {
	if len(code) != 8 && len(code) != 12 && len(code) != 13 && len(code) != 14 {
		return false
	}
	for _, char := range code {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

// firstNonEmpty คืนข้อความแรกที่ไม่ว่าง (ตัดช่องว่างหัวท้าย) ไม่มีเลย = ""
// ใช้เลือกชื่อสินค้า: ชื่อไทย → ชื่อหลัก → ชื่ออังกฤษ
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		text := strings.TrimSpace(value)
		if text != "" {
			return text
		}
	}
	return ""
}

// firstBrand ยี่ห้อแรก (Open Food Facts เก็บหลายยี่ห้อคั่นด้วย , เช่น "Nestlé,Nestle")
func firstBrand(brands string) string {
	brand, _, _ := strings.Cut(brands, ",")
	return strings.TrimSpace(brand)
}
