package cost

import "testing"

func TestCostPrice(t *testing.T) {
	tests := []struct {
		name string
		req  CostSheetRequest
		want float64
	}{
		{
			"วัตถุดิบ 500 + ค่าแรง 300 + อื่น ๆ 200 ผลิต 100 ชิ้น ได้ชิ้นละ 10",
			CostSheetRequest{BatchQuantity: 100, Items: []CostItemRequest{
				{Type: "material", Name: "เมล็ดกาแฟ", Amount: 300},
				{Type: "material", Name: "นม", Amount: 200},
				{Type: "labor", Name: "ค่าแรง", Amount: 300},
				{Type: "overhead", Name: "ค่าไฟ", Amount: 200},
			}},
			10,
		},
		{"ไม่มีรายการต้นทุน ได้ 0", CostSheetRequest{BatchQuantity: 10}, 0},
		{
			"ต้นทุน 100 ผลิต 3 ชิ้น ได้ชิ้นละ 33.33",
			CostSheetRequest{BatchQuantity: 3, Items: []CostItemRequest{{Type: "material", Name: "แป้ง", Amount: 100}}},
			33.33,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := costPrice(tt.req)
			if got != tt.want {
				t.Errorf("costPrice() = %v, ต้องได้ %v", got, tt.want)
			}
		})
	}
}
