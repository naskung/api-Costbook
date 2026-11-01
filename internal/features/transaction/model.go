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

// CategoryMaterial หมวดรายจ่าย "ซื้อวัตถุดิบ" — ไม่หักในกำไร เพราะคิดอยู่ในต้นทุนสินค้าที่ขายแล้ว (ผู้ใช้เลือก 2026-11-01)
const CategoryMaterial = "material"

// Summary สรุปเงินของเดือน
//   - income / expense = เงินเข้า / ออกตามที่บันทึก (รายรับจากเอกสารขายเป็นยอดรวม VAT)
//   - sales = ยอดขายก่อน VAT หลังหักส่วนลด (เอกสารขายที่ไม่ยกเลิก), other_income = รายรับที่บันทึกเอง (ไม่ได้มาจากเอกสารขาย)
//   - revenue = รายได้ = sales + other_income, cost_of_sales = ต้นทุนสินค้าที่ขาย (ต้นทุน ณ วันที่ขาย)
//   - gross_profit = กำไรขั้นต้น = sales − cost_of_sales
//   - material_expense = ซื้อวัตถุดิบ (ไม่หักในกำไร), operating_expense = รายจ่ายที่หักในกำไร = expense − material_expense
//   - profit = กำไรสุทธิ = revenue − cost_of_sales − operating_expense
type Summary struct {
	Month            string  `json:"month"`
	Income           float64 `json:"income"`
	Expense          float64 `json:"expense"`
	Sales            float64 `json:"sales"`
	OtherIncome      float64 `json:"other_income"`
	Revenue          float64 `json:"revenue"`
	CostOfSales      float64 `json:"cost_of_sales"`
	GrossProfit      float64 `json:"gross_profit"`
	MaterialExpense  float64 `json:"material_expense"`
	OperatingExpense float64 `json:"operating_expense"`
	Profit           float64 `json:"profit"`
}

// MonthAmounts ยอดของเดือนที่อ่านจาก DB (ยังไม่ได้คิดรายได้ / กำไร) — หน้าภาพรวมใช้ด้วย
type MonthAmounts struct {
	Income          float64
	Expense         float64
	Sales           float64
	OtherIncome     float64
	CostOfSales     float64
	MaterialExpense float64
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
