package utils

import "testing"

func TestLineAmount(t *testing.T) {
	tests := []struct {
		name      string
		quantity  int
		unitPrice float64
		want      float64
	}{
		{"3 ชิ้น ชิ้นละ 45 ได้ 135", 3, 45, 135},
		{"3 ชิ้น ชิ้นละ 0.1 ได้ 0.3 พอดี", 3, 0.1, 0.3},
		{"ราคา 0 ได้ 0", 5, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LineAmount(tt.quantity, tt.unitPrice)
			if got != tt.want {
				t.Errorf("LineAmount(%d, %v) = %v, ต้องได้ %v", tt.quantity, tt.unitPrice, got, tt.want)
			}
		})
	}
}

func TestDocumentAmounts(t *testing.T) {
	tests := []struct {
		name             string
		itemsTotal       float64
		discount         float64
		withVAT          bool
		pricesIncludeVAT bool
		want             DocAmounts
	}{
		{"ราคาไม่รวม VAT: 100 + VAT 7 = 107", 100, 0, true, false,
			DocAmounts{ItemsTotal: 100, Subtotal: 100, VAT: 7, Total: 107}},
		{"ส่วนลด 10: (100 − 10) + VAT 6.30 = 96.30", 100, 10, true, false,
			DocAmounts{ItemsTotal: 100, Discount: 10, Subtotal: 90, VAT: 6.3, Total: 96.3}},
		{"ราคารวม VAT แล้ว: ยอดสุทธิ 107 ถอดเป็น 100 + VAT 7", 107, 0, true, true,
			DocAmounts{ItemsTotal: 107, Subtotal: 100, VAT: 7, Total: 107}},
		{"ไม่คิด VAT (ร้านไม่จด VAT): ยอดสุทธิ 100 − 5 = 95", 100, 5, false, false,
			DocAmounts{ItemsTotal: 100, Discount: 5, Subtotal: 95, Total: 95}},
		{"VAT คิดจากยอดรวมทั้งใบ ไม่คิดทีละบรรทัด: 0.30 → VAT 0.02", 0.3, 0, true, false,
			DocAmounts{ItemsTotal: 0.3, Subtotal: 0.3, VAT: 0.02, Total: 0.32}},
		{"ส่วนลดเท่ายอดรวม ได้ 0 ทุกช่อง", 100, 100, true, false,
			DocAmounts{ItemsTotal: 100, Discount: 100}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DocumentAmounts(tt.itemsTotal, tt.discount, tt.withVAT, tt.pricesIncludeVAT)
			if got != tt.want {
				t.Errorf("DocumentAmounts() = %+v, ต้องได้ %+v", got, tt.want)
			}
		})
	}
}
