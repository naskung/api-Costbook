package quotation

import "testing"

func TestQuotationRequest_Validate(t *testing.T) {
	const productID = "11111111-1111-1111-1111-111111111111"
	validItem := ItemRequest{ProductID: productID, Quantity: 2, UnitPrice: 45}

	valid := QuotationRequest{IssueDate: "2026-11-21", ValidUntil: "2026-12-21", CustomerName: "บริษัท ลูกค้า จำกัด", Items: []ItemRequest{validItem}}
	sameDay := valid
	sameDay.ValidUntil = "2026-11-21"
	badDate := valid
	badDate.IssueDate = "2026-02-30"
	noValidUntil := valid
	noValidUntil.ValidUntil = ""
	validBeforeIssue := valid
	validBeforeIssue.ValidUntil = "2026-11-20"
	noCustomer := valid
	noCustomer.CustomerName = "  "
	badTaxID := valid
	badTaxID.CustomerTaxID = "123"
	goodTaxID := valid
	goodTaxID.CustomerTaxID = "0105555555555"
	noItems := valid
	noItems.Items = nil
	fullDiscount := valid
	fullDiscount.Discount = 90
	tooMuchDiscount := valid
	tooMuchDiscount.Discount = 90.01
	negativeDiscount := valid
	negativeDiscount.Discount = -1
	zeroQuantity := valid
	zeroQuantity.Items = []ItemRequest{{ProductID: productID}}
	negativePrice := valid
	negativePrice.Items = []ItemRequest{{ProductID: productID, Quantity: 1, UnitPrice: -1}}
	noProductID := valid
	noProductID.Items = []ItemRequest{{Quantity: 1}}

	tests := []struct {
		name string
		req  QuotationRequest
		want string
	}{
		{"ข้อมูลครบ ผ่าน", valid, ""},
		{"ยืนราคาถึงวันเดียวกับวันที่ออก ผ่าน", sameDay, ""},
		{"มีเลขผู้เสียภาษีลูกค้า 13 หลัก ผ่าน", goodTaxID, ""},
		{"ส่วนลดเท่ายอดรวมพอดี ผ่าน", fullDiscount, ""},
		{"วันที่ออกไม่มีจริง ไม่ผ่าน", badDate, "quotation_date_invalid"},
		{"ไม่ส่งวันยืนราคา ไม่ผ่าน", noValidUntil, "quotation_valid_until_invalid"},
		{"ยืนราคาถึงก่อนวันที่ออก ไม่ผ่าน", validBeforeIssue, "quotation_valid_until_invalid"},
		{"ไม่มีชื่อลูกค้า (มีแต่ช่องว่าง) ไม่ผ่าน", noCustomer, "quotation_customer_required"},
		{"เลขผู้เสียภาษีลูกค้าผิด ไม่ผ่าน", badTaxID, "quotation_customer_tax_id_invalid"},
		{"ไม่มีสินค้าในใบ ไม่ผ่าน", noItems, "quotation_items_required"},
		{"สินค้าไม่มี id ไม่ผ่าน", noProductID, "quotation_item_invalid"},
		{"จำนวน 0 ไม่ผ่าน", zeroQuantity, "quotation_item_invalid"},
		{"ราคาติดลบ ไม่ผ่าน", negativePrice, "quotation_item_invalid"},
		{"ส่วนลดมากกว่ายอดรวม ไม่ผ่าน", tooMuchDiscount, "quotation_discount_invalid"},
		{"ส่วนลดติดลบ ไม่ผ่าน", negativeDiscount, "quotation_discount_invalid"},
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
