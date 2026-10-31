package field

import (
	"reflect"
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
