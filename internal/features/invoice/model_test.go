package invoice

import "testing"

func TestInvoiceRequest_Validate(t *testing.T) {
	const productID = "11111111-1111-1111-1111-111111111111"
	validItem := ItemRequest{ProductID: productID, Quantity: 2, UnitPrice: 45}

	valid := InvoiceRequest{IssueDate: "2026-10-31", CustomerName: "บริษัท ลูกค้า จำกัด", Items: []ItemRequest{validItem}}
	withDate := valid
	withDate.IssueDate = "2026-02-30"
	withCustomer := valid
	withCustomer.CustomerName = "  "
	withTaxID := valid
	withTaxID.CustomerTaxID = "123"
	withGoodTaxID := valid
	withGoodTaxID.CustomerTaxID = "0105555555555"
	withNoItems := valid
	withNoItems.Items = nil

	tests := []struct {
		name string
		req  InvoiceRequest
		want string
	}{
		{"ข้อมูลครบ ผ่าน", valid, ""},
		{"มีเลขผู้เสียภาษีลูกค้า 13 หลัก ผ่าน", withGoodTaxID, ""},
		{"วันที่ไม่มีจริง ไม่ผ่าน", withDate, "invoice_date_invalid"},
		{"ไม่มีชื่อลูกค้า ไม่ผ่าน", withCustomer, "invoice_customer_required"},
		{"เลขผู้เสียภาษีลูกค้าผิด ไม่ผ่าน", withTaxID, "invoice_customer_tax_id_invalid"},
		{"ไม่มีสินค้าในใบ ไม่ผ่าน", withNoItems, "invoice_items_required"},
		{"สินค้าไม่มี id ไม่ผ่าน", InvoiceRequest{IssueDate: "2026-10-31", CustomerName: "ลูกค้า", Items: []ItemRequest{{Quantity: 1}}}, "invoice_item_invalid"},
		{"จำนวน 0 ไม่ผ่าน", InvoiceRequest{IssueDate: "2026-10-31", CustomerName: "ลูกค้า", Items: []ItemRequest{{ProductID: productID}}}, "invoice_item_invalid"},
		{"ราคาติดลบ ไม่ผ่าน", InvoiceRequest{IssueDate: "2026-10-31", CustomerName: "ลูกค้า", Items: []ItemRequest{{ProductID: productID, Quantity: 1, UnitPrice: -1}}}, "invoice_item_invalid"},
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
