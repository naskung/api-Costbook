package invoice

import "testing"

func TestInvoiceNumber(t *testing.T) {
	tests := []struct {
		name   string
		year   int
		number int
		want   string
	}{
		{"ปี 2026 ใบแรก", 2026, 1, "INV-2026-0001"},
		{"ปี 2026 ใบที่ 125", 2026, 125, "INV-2026-0125"},
		{"เกิน 9999 ใบ เลขยาวขึ้น ไม่ตัดทิ้ง", 2027, 10000, "INV-2027-10000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := invoiceNumber(tt.year, tt.number)
			if got != tt.want {
				t.Errorf("invoiceNumber(%d, %d) = %q, ต้องได้ %q", tt.year, tt.number, got, tt.want)
			}
		})
	}
}

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
			got := lineAmount(tt.quantity, tt.unitPrice)
			if got != tt.want {
				t.Errorf("lineAmount(%d, %v) = %v, ต้องได้ %v", tt.quantity, tt.unitPrice, got, tt.want)
			}
		})
	}
}

func TestTotals(t *testing.T) {
	tests := []struct {
		name         string
		items        []ItemRequest
		wantSubtotal float64
		wantVAT      float64
		wantTotal    float64
	}{
		{"ยอด 100 บาท VAT 7 รวม 107", []ItemRequest{{Quantity: 2, UnitPrice: 50}}, 100, 7, 107},
		{
			"VAT คิดจากยอดรวมทั้งใบ ไม่คิดทีละบรรทัด: 0.10 + 0.10 + 0.10 = 0.30 → VAT 0.02",
			[]ItemRequest{{Quantity: 1, UnitPrice: 0.1}, {Quantity: 1, UnitPrice: 0.1}, {Quantity: 1, UnitPrice: 0.1}},
			0.3, 0.02, 0.32,
		},
		{"ยอด 1,234.56 VAT 86.42 รวม 1,320.98", []ItemRequest{{Quantity: 1, UnitPrice: 1234.56}}, 1234.56, 86.42, 1320.98},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subtotal, vat, total := totals(tt.items)
			if subtotal != tt.wantSubtotal || vat != tt.wantVAT || total != tt.wantTotal {
				t.Errorf("totals() = (%v, %v, %v), ต้องได้ (%v, %v, %v)", subtotal, vat, total, tt.wantSubtotal, tt.wantVAT, tt.wantTotal)
			}
		})
	}
}
