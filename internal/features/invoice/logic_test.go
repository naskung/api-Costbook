package invoice

import "testing"

func TestDocTypeFor(t *testing.T) {
	if got := docTypeFor(true); got != docTaxInvoice {
		t.Errorf("ร้านจด VAT ได้ %q, ต้องได้ %q", got, docTaxInvoice)
	}
	if got := docTypeFor(false); got != docReceipt {
		t.Errorf("ร้านไม่จด VAT ได้ %q, ต้องได้ %q", got, docReceipt)
	}
}

func TestDeliveryStatusFor(t *testing.T) {
	if got := deliveryStatusFor(true); got != deliveryPending {
		t.Errorf("ร้านติดตามการส่ง ใบใหม่ได้ %q, ต้องได้ %q", got, deliveryPending)
	}
	if got := deliveryStatusFor(false); got != deliveryReceived {
		t.Errorf("ร้านไม่ติดตามการส่ง ใบใหม่ได้ %q, ต้องได้ %q", got, deliveryReceived)
	}
}

func TestIsDeliveryStatus(t *testing.T) {
	for _, status := range []string{deliveryPending, deliveryShipped, deliveryReceived} {
		if !isDeliveryStatus(status) {
			t.Errorf("%q ต้องเป็นสถานะที่ใช้ได้", status)
		}
	}
	for _, status := range []string{"", "lost", "Shipped"} {
		if isDeliveryStatus(status) {
			t.Errorf("%q ต้องใช้ไม่ได้", status)
		}
	}
}

func TestInvoiceNumber(t *testing.T) {
	tests := []struct {
		name    string
		docType string
		year    int
		number  int
		want    string
	}{
		{"ใบกำกับปี 2026 ใบแรก", docTaxInvoice, 2026, 1, "INV-2026-0001"},
		{"ใบเสร็จปี 2026 ใบที่ 125", docReceipt, 2026, 125, "RC-2026-0125"},
		{"เกิน 9999 ใบ เลขยาวขึ้น ไม่ตัดทิ้ง", docTaxInvoice, 2027, 10000, "INV-2027-10000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := invoiceNumber(docPrefix(tt.docType), tt.year, tt.number)
			if got != tt.want {
				t.Errorf("invoiceNumber() = %q, ต้องได้ %q", got, tt.want)
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

func TestDocumentAmounts(t *testing.T) {
	twoFifty := []ItemRequest{{Quantity: 2, UnitPrice: 50}}
	pennies := []ItemRequest{{Quantity: 1, UnitPrice: 0.1}, {Quantity: 1, UnitPrice: 0.1}, {Quantity: 1, UnitPrice: 0.1}}

	tests := []struct {
		name             string
		items            []ItemRequest
		discount         float64
		docType          string
		pricesIncludeVAT bool
		want             money
	}{
		{"ใบกำกับ ราคาไม่รวม VAT: 100 + VAT 7 = 107", twoFifty, 0, docTaxInvoice, false,
			money{itemsTotal: 100, subtotal: 100, vat: 7, total: 107}},
		{"ใบกำกับ ส่วนลด 10: (100 − 10) + VAT 6.30 = 96.30", twoFifty, 10, docTaxInvoice, false,
			money{itemsTotal: 100, discount: 10, subtotal: 90, vat: 6.3, total: 96.3}},
		{"ใบกำกับ ราคารวม VAT แล้ว: ยอดสุทธิ 107 ถอดเป็น 100 + VAT 7", []ItemRequest{{Quantity: 1, UnitPrice: 107}}, 0, docTaxInvoice, true,
			money{itemsTotal: 107, subtotal: 100, vat: 7, total: 107}},
		{"ใบเสร็จ (ไม่จด VAT): ไม่มี VAT ยอดสุทธิ 100 − 5 = 95", twoFifty, 5, docReceipt, false,
			money{itemsTotal: 100, discount: 5, subtotal: 95, total: 95}},
		{"VAT คิดจากยอดรวมทั้งใบ ไม่คิดทีละบรรทัด: 0.30 → VAT 0.02", pennies, 0, docTaxInvoice, false,
			money{itemsTotal: 0.3, subtotal: 0.3, vat: 0.02, total: 0.32}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := documentAmounts(tt.items, tt.discount, tt.docType, tt.pricesIncludeVAT)
			if got != tt.want {
				t.Errorf("documentAmounts() = %+v, ต้องได้ %+v", got, tt.want)
			}
		})
	}
}
