package barcode

import "testing"

func TestIsBarcode(t *testing.T) {
	tests := []struct {
		name string
		code string
		want bool
	}{
		{"EAN-13 ของไทย ผ่าน", "8850124003850", true},
		{"EAN-8 ผ่าน", "96385074", true},
		{"UPC-A 12 หลัก ผ่าน", "036000291452", true},
		{"GTIN-14 ผ่าน", "18850124003857", true},
		{"ว่าง ไม่ผ่าน", "", false},
		{"มีตัวอักษร ไม่ผ่าน (รหัสที่ร้านตั้งเอง เช่น CF001)", "CF001", false},
		{"ตัวเลข 10 หลัก ไม่ผ่าน", "1234567890", false},
		{"มีช่องว่าง ไม่ผ่าน", "885012400385 ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBarcode(tt.code); got != tt.want {
				t.Errorf("isBarcode(%q) = %v, ต้องได้ %v", tt.code, got, tt.want)
			}
		})
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "  ", " น้ำดื่ม ", "Water"); got != "น้ำดื่ม" {
		t.Errorf("ต้องได้ข้อความแรกที่ไม่ว่าง (ตัดช่องว่าง) ได้ %q", got)
	}
	if got := firstNonEmpty("", " "); got != "" {
		t.Errorf("ว่างทั้งหมด ต้องได้ \"\" ได้ %q", got)
	}
}

func TestFirstBrand(t *testing.T) {
	if got := firstBrand("Nestlé, Nestle"); got != "Nestlé" {
		t.Errorf("หลายยี่ห้อ ต้องได้ตัวแรก ได้ %q", got)
	}
	if got := firstBrand(""); got != "" {
		t.Errorf("ไม่มียี่ห้อ ต้องได้ \"\" ได้ %q", got)
	}
}
