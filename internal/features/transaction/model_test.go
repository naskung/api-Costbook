package transaction

import "testing"

func TestTransactionRequest_Validate(t *testing.T) {
	valid := TransactionRequest{Type: "expense", Category: "material", Amount: 250.5, Date: "2026-10-31"}

	withType := valid
	withType.Type = "refund"
	withCategory := valid
	withCategory.Category = " "
	withZero := valid
	withZero.Amount = 0
	withNegative := valid
	withNegative.Amount = -10
	withBadDate := valid
	withBadDate.Date = "31/10/2026"

	tests := []struct {
		name string
		req  TransactionRequest
		want string
	}{
		{"ข้อมูลครบ ผ่าน", valid, ""},
		{"ประเภทผิด ไม่ผ่าน", withType, "transaction_type_invalid"},
		{"ไม่มีหมวด ไม่ผ่าน", withCategory, "transaction_category_required"},
		{"จำนวนเงิน 0 ไม่ผ่าน", withZero, "transaction_amount_invalid"},
		{"จำนวนเงินติดลบ ไม่ผ่าน", withNegative, "transaction_amount_invalid"},
		{"วันที่ผิดรูปแบบ ไม่ผ่าน", withBadDate, "transaction_date_invalid"},
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
