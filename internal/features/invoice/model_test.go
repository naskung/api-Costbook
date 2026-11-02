package invoice

import "testing"

func TestInvoiceRequest_Validate(t *testing.T) {
	const productID = "11111111-1111-1111-1111-111111111111"
	validItem := ItemRequest{ProductID: productID, Quantity: 2, UnitPrice: 45}

	valid := InvoiceRequest{IssueDate: "2026-10-31", CustomerName: "บริษัท ลูกค้า จำกัด", Items: []ItemRequest{validItem}}
	noCustomer := valid
	noCustomer.CustomerName = ""
	withDate := valid
	withDate.IssueDate = "2026-02-30"
	withTaxID := valid
	withTaxID.CustomerTaxID = "123"
	withGoodTaxID := valid
	withGoodTaxID.CustomerTaxID = "0105555555555"
	withNoItems := valid
	withNoItems.Items = nil
	withDiscount := valid
	withDiscount.Discount = 90
	tooMuchDiscount := valid
	tooMuchDiscount.Discount = 90.01
	negativeDiscount := valid
	negativeDiscount.Discount = -1

	tests := []struct {
		name string
		req  InvoiceRequest
		want string
	}{
		{"ข้อมูลครบ ผ่าน", valid, ""},
		{"ไม่มีชื่อลูกค้า ผ่าน (ใบเสร็จไม่ต้องมี — ใบกำกับเช็คตอนออกใบ)", noCustomer, ""},
		{"มีเลขผู้เสียภาษีลูกค้า 13 หลัก ผ่าน", withGoodTaxID, ""},
		{"ส่วนลดเท่ายอดรวมพอดี ผ่าน", withDiscount, ""},
		{"วันที่ไม่มีจริง ไม่ผ่าน", withDate, "invoice_date_invalid"},
		{"เลขผู้เสียภาษีลูกค้าผิด ไม่ผ่าน", withTaxID, "invoice_customer_tax_id_invalid"},
		{"ไม่มีสินค้าในใบ ไม่ผ่าน", withNoItems, "invoice_items_required"},
		{"สินค้าไม่มี id ไม่ผ่าน", InvoiceRequest{IssueDate: "2026-10-31", Items: []ItemRequest{{Quantity: 1}}}, "invoice_item_invalid"},
		{"จำนวน 0 ไม่ผ่าน", InvoiceRequest{IssueDate: "2026-10-31", Items: []ItemRequest{{ProductID: productID}}}, "invoice_item_invalid"},
		{"ราคาติดลบ ไม่ผ่าน", InvoiceRequest{IssueDate: "2026-10-31", Items: []ItemRequest{{ProductID: productID, Quantity: 1, UnitPrice: -1}}}, "invoice_item_invalid"},
		{"ส่วนลดมากกว่ายอดรวม ไม่ผ่าน", tooMuchDiscount, "invoice_discount_invalid"},
		{"ส่วนลดติดลบ ไม่ผ่าน", negativeDiscount, "invoice_discount_invalid"},
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

func TestDeliveryRequest_Validate(t *testing.T) {
	if got := (DeliveryRequest{DeliveryStatus: "received"}).Validate(); got != "" {
		t.Errorf("received ต้องผ่าน ได้ %q", got)
	}
	if got := (DeliveryRequest{DeliveryStatus: "lost"}).Validate(); got != "invoice_delivery_status_invalid" {
		t.Errorf("สถานะที่ไม่มี ต้องได้ invoice_delivery_status_invalid ได้ %q", got)
	}
}
