package transaction

import (
	"strings"

	"github.com/naskung/api-Costbook/internal/utils"
)

// Transaction รายรับ / รายจ่าย 1 รายการ
// type: income = รายรับ, expense = รายจ่าย / category = รหัสหมวด เช่น sales, material (Flutter แปลเป็นชื่อหมวด)
// invoice_no ไม่ว่าง = รายรับจากใบกำกับ แก้ / ลบตรง ๆ ไม่ได้
type Transaction struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"`
	Category  string  `json:"category"`
	Amount    float64 `json:"amount"`
	Date      string  `json:"date"`
	Note      string  `json:"note"`
	InvoiceID string  `json:"invoice_id"`
	InvoiceNo string  `json:"invoice_no"`
}

// TransactionRequest ข้อมูลที่รับมาตอนเพิ่ม / แก้รายการ
type TransactionRequest struct {
	Type     string  `json:"type"`
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
	Date     string  `json:"date"`
	Note     string  `json:"note"`
}

// Summary สรุปรายรับ รายจ่าย กำไร ของเดือน
type Summary struct {
	Month   string  `json:"month"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
	Profit  float64 `json:"profit"`
}

// Validate เช็คข้อมูลก่อนบันทึก คืน error code ("" = ผ่าน)
func (req TransactionRequest) Validate() string {
	if req.Type != "income" && req.Type != "expense" {
		return "transaction_type_invalid"
	}
	if strings.TrimSpace(req.Category) == "" {
		return "transaction_category_required"
	}
	if req.Amount <= 0 {
		return "transaction_amount_invalid"
	}
	if !utils.IsDate(req.Date) {
		return "transaction_date_invalid"
	}
	return ""
}
