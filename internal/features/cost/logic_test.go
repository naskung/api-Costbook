package cost

import (
	"reflect"
	"testing"
)

func TestItemAmounts(t *testing.T) {
	tests := []struct {
		name  string
		items []CostItemRequest
		want  []float64
	}{
		{
			"3D print: PLA 40 กรัม × 0.45 + ค่าเครื่อง 2.5 ชม. × 6 + เผื่อเสีย 10%",
			[]CostItemRequest{
				{Type: "material", Kind: "fixed", Quantity: 40, UnitPrice: 0.45},
				{Type: "overhead", Kind: "fixed", Quantity: 2.5, UnitPrice: 6},
				{Type: "overhead", Kind: "percent", Percent: 10},
			},
			[]float64{18, 15, 3.3},
		},
		{"ร้านซื้อมาขายไป: 1 ขวด × 5.20", []CostItemRequest{{Type: "material", Kind: "fixed", Quantity: 1, UnitPrice: 5.2}}, []float64{5.2}},
		{"มีแต่บรรทัด % ได้ 0 (ไม่มีอะไรให้คิด %)", []CostItemRequest{{Type: "overhead", Kind: "percent", Percent: 10}}, []float64{0}},
		{"ไม่มีรายการ ได้รายการว่าง", []CostItemRequest{}, []float64{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := itemAmounts(tt.items)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("itemAmounts() = %v, ต้องได้ %v", got, tt.want)
			}
		})
	}
}

func TestCostPrice(t *testing.T) {
	tests := []struct {
		name  string
		items []CostItemRequest
		batch int
		want  float64
	}{
		{
			"วัตถุดิบ 500 + ค่าแรง 300 + อื่น ๆ 200 ผลิต 100 ชิ้น ได้ชิ้นละ 10",
			[]CostItemRequest{
				{Type: "material", Kind: "fixed", Quantity: 1, UnitPrice: 500},
				{Type: "labor", Kind: "fixed", Quantity: 1, UnitPrice: 300},
				{Type: "overhead", Kind: "fixed", Quantity: 1, UnitPrice: 200},
			},
			100, 10,
		},
		{
			"รวม 36.30 (18 + 15 + 3.30) พิมพ์ได้ 1 ชิ้น ได้ชิ้นละ 36.30",
			[]CostItemRequest{
				{Type: "material", Kind: "fixed", Quantity: 40, UnitPrice: 0.45},
				{Type: "overhead", Kind: "fixed", Quantity: 2.5, UnitPrice: 6},
				{Type: "overhead", Kind: "percent", Percent: 10},
			},
			1, 36.3,
		},
		{"ไม่มีรายการต้นทุน ได้ 0", []CostItemRequest{}, 10, 0},
		{"ต้นทุน 100 ผลิต 3 ชิ้น ได้ชิ้นละ 33.33", []CostItemRequest{{Type: "material", Kind: "fixed", Quantity: 1, UnitPrice: 100}}, 3, 33.33},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := costPrice(tt.items, itemAmounts(tt.items), tt.batch)
			if got != tt.want {
				t.Errorf("costPrice() = %v, ต้องได้ %v", got, tt.want)
			}
		})
	}
}
