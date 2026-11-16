package field

import (
	"reflect"
	"slices"
	"testing"
)

func TestFieldRequest_Validate(t *testing.T) {
	tests := []struct {
		name string
		req  FieldRequest
		want string
	}{
		{"ช่องข้อความ ผ่าน", FieldRequest{Name: "วัสดุ", Type: "text"}, ""},
		{"ช่องตัวเลือกมีตัวเลือก ผ่าน", FieldRequest{Name: "สี", Type: "select", Options: []string{"ขาว", "ดำ"}}, ""},
		{"ช่องวันที่ ผ่าน", FieldRequest{Name: "วันหมดอายุ", Type: "date"}, ""},
		{"ชื่อว่าง ไม่ผ่าน", FieldRequest{Name: " ", Type: "text"}, "field_name_required"},
		{"ชนิดผิด ไม่ผ่าน", FieldRequest{Name: "สี", Type: "color"}, "field_type_invalid"},
		{"ช่องตัวเลือกไม่มีตัวเลือก ไม่ผ่าน", FieldRequest{Name: "สี", Type: "select"}, "field_options_required"},
		{"มีตัวเลือกว่าง ไม่ผ่าน", FieldRequest{Name: "สี", Type: "select", Options: []string{"ขาว", ""}}, "field_options_required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.req.Validate()
			if got != tt.want {
				t.Errorf("Validate() = %q, ต้องได้ %q", got, tt.want)
			}
		})
	}
}

func TestFieldRequest_CleanOptions(t *testing.T) {
	tests := []struct {
		name string
		req  FieldRequest
		want []string
	}{
		{"ช่องตัวเลือก เก็บตัวเลือกไว้", FieldRequest{Type: "select", Options: []string{"ขาว"}}, []string{"ขาว"}},
		{"ช่องข้อความ ไม่เก็บตัวเลือก", FieldRequest{Type: "text", Options: []string{"ขาว"}}, []string{}},
		{"ไม่ส่งตัวเลือกมา ได้รายการว่าง ไม่ใช่ nil", FieldRequest{Type: "select"}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.req.cleanOptions()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("cleanOptions() = %#v, ต้องได้ %#v", got, tt.want)
			}
		})
	}
}

func TestLayoutRequest_Validate(t *testing.T) {
	allBlocks := []LayoutItem{}
	for i, block := range FormBlocks {
		allBlocks = append(allBlocks, LayoutItem{Key: block, SortOrder: i + 1})
	}
	withField := append(slices.Clone(allBlocks), LayoutItem{Key: "11111111-1111-1111-1111-111111111111", SortOrder: 2})

	tests := []struct {
		name  string
		items []LayoutItem
		want  string
	}{
		{"ช่องหลักครบ ผ่าน", allBlocks, ""},
		{"ช่องที่ร้านเพิ่มแทรกตรงไหนก็ได้ (เลขซ้ำก็ได้) ผ่าน", withField, ""},
		{"ไม่มีอะไรเลย", nil, "form_layout_invalid"},
		{"ขาดช่องหลัก", allBlocks[1:], "form_layout_invalid"},
		{"key ซ้ำ", append(slices.Clone(allBlocks), LayoutItem{Key: BlockName, SortOrder: 9}), "form_layout_invalid"},
		{"key ไม่รู้จัก (ไม่ใช่ช่องหลัก ไม่ใช่ UUID)", append(slices.Clone(allBlocks), LayoutItem{Key: "price", SortOrder: 9}), "form_layout_invalid"},
		{"sort_order ติดลบ", append(slices.Clone(allBlocks[1:]), LayoutItem{Key: BlockImage, SortOrder: -1}), "form_layout_invalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := (LayoutRequest{Items: tt.items}).Validate(); got != tt.want {
				t.Errorf("ได้ %q ต้องได้ %q", got, tt.want)
			}
		})
	}
}
