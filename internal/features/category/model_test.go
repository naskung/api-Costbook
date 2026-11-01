package category

import (
	"strings"
	"testing"
)

func TestCategoryRequest_Validate(t *testing.T) {
	tests := []struct {
		name string
		req  CategoryRequest
		want string
	}{
		{"ชื่อปกติ ผ่าน", CategoryRequest{Name: "งานพิมพ์ 3D"}, ""},
		{"ชื่อว่าง ไม่ผ่าน", CategoryRequest{Name: ""}, "category_name_required"},
		{"มีแต่ช่องว่าง ไม่ผ่าน", CategoryRequest{Name: "   "}, "category_name_required"},
		{"ภาษาไทย 100 ตัว ผ่าน (นับตัวอักษร ไม่ใช่ไบต์)", CategoryRequest{Name: strings.Repeat("ก", 100)}, ""},
		{"ยาวเกิน 100 ตัว ไม่ผ่าน", CategoryRequest{Name: strings.Repeat("ก", 101)}, "category_name_too_long"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.req.Validate(); got != tt.want {
				t.Errorf("Validate() = %q, ต้องได้ %q", got, tt.want)
			}
		})
	}
}

func TestCategoryRequest_CleanName(t *testing.T) {
	if got := (CategoryRequest{Name: "  พวงกุญแจ  "}).cleanName(); got != "พวงกุญแจ" {
		t.Errorf("cleanName() = %q, ต้องได้ %q", got, "พวงกุญแจ")
	}
}
