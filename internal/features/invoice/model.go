package invoice

import (
	"strings"

	"github.com/naskung/api-Costbook/internal/utils"
)

// Invoice ใบกำกับภาษี 1 ใบ (Flutter เอาไปแสดงและสร้าง PDF)
// ข้อมูลผู้ขาย / ผู้ซื้อ / ชื่อสินค้า เป็นค่า ณ วันที่ออก แก้ข้อมูลร้านหรือสินค้าทีหลัง ใบนี้ไม่เปลี่ยน
// status: issued = ออกแล้ว, cancelled = ยกเลิกแล้ว
type Invoice struct {
	ID              string        `json:"id"`
	InvoiceNo       string        `json:"invoice_no"`
	IssueDate       string        `json:"issue_date"`
	Status          string        `json:"status"`
	SellerName      string        `json:"seller_name"`
	SellerAddress   string        `json:"seller_address"`
	SellerTaxID     string        `json:"seller_tax_id"`
	SellerBranch    string        `json:"seller_branch"`
	SellerPhone     string        `json:"seller_phone"`
	CustomerName    string        `json:"customer_name"`
	CustomerAddress string        `json:"customer_address"`
	CustomerTaxID   string        `json:"customer_tax_id"`
	CustomerBranch  string        `json:"customer_branch"`
	Note            string        `json:"note"`
	Subtotal        float64       `json:"subtotal"`
	VAT             float64       `json:"vat"`
	Total           float64       `json:"total"`
	Items           []InvoiceItem `json:"items"`
}

// InvoiceItem สินค้า 1 บรรทัดในใบกำกับ (amount = quantity × unit_price)
type InvoiceItem struct {
	ID        string  `json:"id"`
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Unit      string  `json:"unit"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Amount    float64 `json:"amount"`
}

// InvoiceSummary ข้อมูลย่อของใบกำกับ ใช้ในหน้ารายการ
type InvoiceSummary struct {
	ID           string  `json:"id"`
	InvoiceNo    string  `json:"invoice_no"`
	IssueDate    string  `json:"issue_date"`
	Status       string  `json:"status"`
	CustomerName string  `json:"customer_name"`
	Total        float64 `json:"total"`
}

// InvoiceRequest ข้อมูลที่รับมาตอนออกใบกำกับ (ราคาต่อหน่วยเป็นราคาก่อน VAT)
type InvoiceRequest struct {
	IssueDate       string        `json:"issue_date"`
	CustomerName    string        `json:"customer_name"`
	CustomerAddress string        `json:"customer_address"`
	CustomerTaxID   string        `json:"customer_tax_id"`
	CustomerBranch  string        `json:"customer_branch"`
	Note            string        `json:"note"`
	Items           []ItemRequest `json:"items"`
}

type ItemRequest struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

// Validate เช็คข้อมูลก่อนออกใบ คืน error code ("" = ผ่าน)
func (req InvoiceRequest) Validate() string {
	if !utils.IsDate(req.IssueDate) {
		return "invoice_date_invalid"
	}
	if strings.TrimSpace(req.CustomerName) == "" {
		return "invoice_customer_required"
	}
	if req.CustomerTaxID != "" && !utils.IsTaxID(req.CustomerTaxID) {
		return "invoice_customer_tax_id_invalid"
	}
	if len(req.Items) == 0 {
		return "invoice_items_required"
	}
	for _, item := range req.Items {
		if !utils.IsUUID(item.ProductID) || item.Quantity <= 0 || item.UnitPrice < 0 {
			return "invoice_item_invalid"
		}
	}
	return ""
}
