package stock

import "testing"

func TestMovementRequest_Validate(t *testing.T) {
	const productID = "11111111-1111-1111-1111-111111111111"

	tests := []struct {
		name string
		req  MovementRequest
		want string
	}{
		{"ข้อมูลครบ ผ่าน", MovementRequest{ProductID: productID, Quantity: 10, Note: "ซื้อเพิ่ม"}, ""},
		{"ไม่ได้เลือกสินค้า ไม่ผ่าน", MovementRequest{Quantity: 10}, "stock_product_required"},
		{"id สินค้าผิดรูปแบบ ไม่ผ่าน", MovementRequest{ProductID: "abc", Quantity: 10}, "stock_product_required"},
		{"จำนวน 0 ไม่ผ่าน", MovementRequest{ProductID: productID, Quantity: 0}, "stock_quantity_invalid"},
		{"จำนวนติดลบ ไม่ผ่าน", MovementRequest{ProductID: productID, Quantity: -3}, "stock_quantity_invalid"},
		{"รับเข้าพร้อมราคาทุน ผ่าน", MovementRequest{ProductID: productID, Quantity: 10, UnitCost: 5.2, Supplier: "ร้านส่ง"}, ""},
		{"ราคาทุนติดลบ ไม่ผ่าน", MovementRequest{ProductID: productID, Quantity: 10, UnitCost: -1}, "stock_unit_cost_invalid"},
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
