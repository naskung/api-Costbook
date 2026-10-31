package cost

import "testing"

func TestCostSheetRequest_Validate(t *testing.T) {
	validItem := CostItemRequest{Type: "material", Kind: "fixed", Name: "เส้น PLA", Quantity: 40, Unit: "กรัม", UnitPrice: 0.45}

	tests := []struct {
		name string
		req  CostSheetRequest
		want string
	}{
		{"ข้อมูลครบ ผ่าน", CostSheetRequest{BatchQuantity: 10, SellPrice: 20, Items: []CostItemRequest{validItem}}, ""},
		{"บรรทัดแบบ % ผ่าน", CostSheetRequest{BatchQuantity: 1, Items: []CostItemRequest{{Type: "overhead", Kind: "percent", Name: "เผื่อเสีย", Percent: 10}}}, ""},
		{"ยังไม่มีรายการต้นทุน ผ่าน", CostSheetRequest{BatchQuantity: 1}, ""},
		{"จำนวนผลิต 0 ไม่ผ่าน", CostSheetRequest{BatchQuantity: 0}, "cost_batch_quantity_invalid"},
		{"ราคาขายติดลบ ไม่ผ่าน", CostSheetRequest{BatchQuantity: 1, SellPrice: -1}, "product_price_invalid"},
		{"ประเภทต้นทุนผิด ไม่ผ่าน", CostSheetRequest{BatchQuantity: 1, Items: []CostItemRequest{{Type: "tax", Kind: "fixed", Name: "ภาษี"}}}, "cost_type_invalid"},
		{"วิธีคิดผิด ไม่ผ่าน", CostSheetRequest{BatchQuantity: 1, Items: []CostItemRequest{{Type: "labor", Kind: "hour", Name: "ค่าแรง"}}}, "cost_kind_invalid"},
		{"ชื่อรายการว่าง ไม่ผ่าน", CostSheetRequest{BatchQuantity: 1, Items: []CostItemRequest{{Type: "labor", Kind: "fixed", Name: " "}}}, "cost_name_required"},
		{"จำนวนติดลบ ไม่ผ่าน", CostSheetRequest{BatchQuantity: 1, Items: []CostItemRequest{{Type: "overhead", Kind: "fixed", Name: "ค่าไฟ", Quantity: -1}}}, "cost_amount_invalid"},
		{"ราคาต่อหน่วยติดลบ ไม่ผ่าน", CostSheetRequest{BatchQuantity: 1, Items: []CostItemRequest{{Type: "overhead", Kind: "fixed", Name: "ค่าไฟ", UnitPrice: -5}}}, "cost_amount_invalid"},
		{"% เกิน 1000 ไม่ผ่าน", CostSheetRequest{BatchQuantity: 1, Items: []CostItemRequest{{Type: "overhead", Kind: "percent", Name: "เผื่อเสีย", Percent: 1001}}}, "cost_percent_invalid"},
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

func TestPresetRequest_Validate(t *testing.T) {
	tests := []struct {
		name string
		req  PresetRequest
		want string
	}{
		{"เส้น PLA กรัมละ 0.45 ผ่าน", PresetRequest{Type: "material", Kind: "fixed", Name: "เส้น PLA", Unit: "กรัม", UnitPrice: 0.45}, ""},
		{"เผื่อเสีย 10% ผ่าน", PresetRequest{Type: "overhead", Kind: "percent", Name: "เผื่อเสีย", Percent: 10}, ""},
		{"ไม่มีชื่อ ไม่ผ่าน", PresetRequest{Type: "material", Kind: "fixed"}, "cost_name_required"},
		{"ประเภทผิด ไม่ผ่าน", PresetRequest{Type: "x", Kind: "fixed", Name: "a"}, "cost_type_invalid"},
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
