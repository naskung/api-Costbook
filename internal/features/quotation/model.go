package quotation

import (
	"strings"

	"github.com/naskung/api-Costbook/internal/utils"
)

// Quotation ใบเสนอราคา 1 ใบ (Flutter เอาไปแสดงและสร้าง PDF)
// ไม่ตัดสต็อก ไม่บันทึกรายรับ — แค่เสนอราคาให้ลูกค้าก่อนขาย
// ข้อมูลผู้ขาย / ลูกค้า / ชื่อสินค้า เป็นค่า ณ วันที่ออก แก้ข้อมูลร้านหรือสินค้าทีหลัง ใบนี้ไม่เปลี่ยน
// status: issued = ออกแล้ว, cancelled = ยกเลิกแล้ว / valid_until = ยืนราคาถึงวันไหน
// with_vat = ตอนออกร้านจด VAT ไหม (false = ไม่มี VAT ในใบ)
// เงิน: items_total = รวมราคาสินค้า, discount = ส่วนลดทั้งบิล, subtotal = มูลค่าก่อน VAT, vat, total = ยอดสุทธิ
type Quotation struct {
	ID               string          `json:"id"`
	QuotationNo      string          `json:"quotation_no"`
	IssueDate        string          `json:"issue_date"`
	ValidUntil       string          `json:"valid_until"`
	Status           string          `json:"status"`
	SellerName       string          `json:"seller_name"`
	SellerAddress    string          `json:"seller_address"`
	SellerTaxID      string          `json:"seller_tax_id"`
	SellerBranch     string          `json:"seller_branch"`
	SellerPhone      string          `json:"seller_phone"`
	CustomerName     string          `json:"customer_name"`
	CustomerAddress  string          `json:"customer_address"`
	CustomerTaxID    string          `json:"customer_tax_id"`
	CustomerBranch   string          `json:"customer_branch"`
	Note             string          `json:"note"`
	WithVAT          bool            `json:"with_vat"`
	PricesIncludeVAT bool            `json:"prices_include_vat"`
	ItemsTotal       float64         `json:"items_total"`
	Discount         float64         `json:"discount"`
	Subtotal         float64         `json:"subtotal"`
	VAT              float64         `json:"vat"`
	Total            float64         `json:"total"`
	Items            []QuotationItem `json:"items"`
}

// QuotationItem สินค้า 1 บรรทัดในใบ (amount = quantity × unit_price)
// product_id = "" ถ้าสินค้าถูกลบไปแล้ว (ใบยังมีชื่อ / ราคาครบ)
type QuotationItem struct {
	ID        string  `json:"id"`
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Unit      string  `json:"unit"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Amount    float64 `json:"amount"`
}

// QuotationSummary ข้อมูลย่อของใบ ใช้ในหน้ารายการ
type QuotationSummary struct {
	ID           string  `json:"id"`
	QuotationNo  string  `json:"quotation_no"`
	IssueDate    string  `json:"issue_date"`
	ValidUntil   string  `json:"valid_until"`
	Status       string  `json:"status"`
	CustomerName string  `json:"customer_name"`
	Total        float64 `json:"total"`
}

// QuotationRequest ข้อมูลที่รับมาตอนออกใบเสนอราคา
// คิด VAT ไหม / ราคารวม VAT ไหม API เลือกเองจากรูปแบบร้าน (ตั้งค่าร้าน) ไม่ต้องส่งมา
type QuotationRequest struct {
	IssueDate       string        `json:"issue_date"`
	ValidUntil      string        `json:"valid_until"`
	CustomerName    string        `json:"customer_name"`
	CustomerAddress string        `json:"customer_address"`
	CustomerTaxID   string        `json:"customer_tax_id"`
	CustomerBranch  string        `json:"customer_branch"`
	Note            string        `json:"note"`
	Discount        float64       `json:"discount"`
	Items           []ItemRequest `json:"items"`
}

type ItemRequest struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

// Validate เช็คข้อมูลก่อนออกใบ คืน error code ("" = ผ่าน)
// วันที่เป็นแบบ 2026-11-21 → เทียบเป็นข้อความได้เลยว่าวันไหนก่อน
func (req QuotationRequest) Validate() string {
	if !utils.IsDate(req.IssueDate) {
		return "quotation_date_invalid"
	}
	if !utils.IsDate(req.ValidUntil) || req.ValidUntil < req.IssueDate {
		return "quotation_valid_until_invalid"
	}
	if strings.TrimSpace(req.CustomerName) == "" {
		return "quotation_customer_required"
	}
	if req.CustomerTaxID != "" && !utils.IsTaxID(req.CustomerTaxID) {
		return "quotation_customer_tax_id_invalid"
	}
	if len(req.Items) == 0 {
		return "quotation_items_required"
	}
	for _, item := range req.Items {
		if !utils.IsUUID(item.ProductID) || item.Quantity <= 0 || item.UnitPrice < 0 {
			return "quotation_item_invalid"
		}
	}
	if req.Discount < 0 || req.Discount > itemsTotal(req.Items) {
		return "quotation_discount_invalid"
	}
	return ""
}
