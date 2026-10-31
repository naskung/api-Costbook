package cost

import "testing"

func TestCostSheetRequest_Validate(t *testing.T) {
	validItem := CostItemRequest{Type: "material", Name: "แป้ง", Amount: 100}

	tests := []struct {
		name string
		req  CostSheetRequest
		want string
	}{
		{"ข้อมูลครบ ผ่าน", CostSheetRequest{BatchQuantity: 10, SellPrice: 20, Items: []CostItemRequest{validItem}}, ""},
		{"ยังไม่มีรายการต้นทุน ผ่าน", CostSheetRequest{BatchQuantity: 1}, ""},
		{"จำนวนผลิต 0 ไม่ผ่าน", CostSheetRequest{BatchQuantity: 0}, "cost_batch_quantity_invalid"},
		{"ราคาขายติดลบ ไม่ผ่าน", CostSheetRequest{BatchQuantity: 1, SellPrice: -1}, "product_price_invalid"},
		{"ประเภทต้นทุนผิด ไม่ผ่าน", CostSheetRequest{BatchQuantity: 1, Items: []CostItemRequest{{Type: "tax", Name: "ภาษี", Amount: 1}}}, "cost_type_invalid"},
		{"ชื่อรายการว่าง ไม่ผ่าน", CostSheetRequest{BatchQuantity: 1, Items: []CostItemRequest{{Type: "labor", Name: " ", Amount: 1}}}, "cost_name_required"},
		{"จำนวนเงินติดลบ ไม่ผ่าน", CostSheetRequest{BatchQuantity: 1, Items: []CostItemRequest{{Type: "overhead", Name: "ค่าไฟ", Amount: -5}}}, "cost_amount_invalid"},
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
