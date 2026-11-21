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
