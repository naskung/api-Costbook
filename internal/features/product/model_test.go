package product

import "testing"

func TestProductRequest_Validate(t *testing.T) {
	tests := []struct {
		name string
		req  ProductRequest
		want string
	}{
		{"ข้อมูลครบ ผ่าน", ProductRequest{Name: "กาแฟ", Unit: "แก้ว", CostPrice: 20, SellPrice: 45, MinStock: 5}, ""},
		{"ราคาเป็น 0 ผ่าน", ProductRequest{Name: "กาแฟ"}, ""},
		{"ชื่อว่าง ไม่ผ่าน", ProductRequest{Name: "", SellPrice: 45}, "product_name_required"},
		{"ชื่อมีแต่ช่องว่าง ไม่ผ่าน", ProductRequest{Name: "   ", SellPrice: 45}, "product_name_required"},
		{"ต้นทุนติดลบ ไม่ผ่าน", ProductRequest{Name: "กาแฟ", CostPrice: -1}, "product_price_invalid"},
		{"ราคาขายติดลบ ไม่ผ่าน", ProductRequest{Name: "กาแฟ", SellPrice: -0.5}, "product_price_invalid"},
		{"จำนวนขั้นต่ำติดลบ ไม่ผ่าน", ProductRequest{Name: "กาแฟ", MinStock: -1}, "product_min_stock_invalid"},
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
