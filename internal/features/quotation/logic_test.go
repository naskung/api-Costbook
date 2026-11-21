package quotation

import "testing"

func TestQuotationNumber(t *testing.T) {
	tests := []struct {
		name   string
		year   int
		number int
		want   string
	}{
		{"ปี 2026 ใบแรก", 2026, 1, "QT-2026-0001"},
		{"ปี 2026 ใบที่ 125", 2026, 125, "QT-2026-0125"},
		{"เกิน 9999 ใบ เลขยาวขึ้น ไม่ตัดทิ้ง", 2027, 10000, "QT-2027-10000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := quotationNumber(tt.year, tt.number)
			if got != tt.want {
				t.Errorf("quotationNumber() = %q, ต้องได้ %q", got, tt.want)
			}
		})
	}
}

func TestItemsTotal(t *testing.T) {
	items := []ItemRequest{{Quantity: 2, UnitPrice: 50}, {Quantity: 3, UnitPrice: 0.1}}
	if got := itemsTotal(items); got != 100.3 {
		t.Errorf("itemsTotal() = %v, ต้องได้ 100.3", got)
	}
	if got := itemsTotal(nil); got != 0 {
		t.Errorf("ไม่มีสินค้า ต้องได้ 0 ได้ %v", got)
	}
}
