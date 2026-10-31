package invoice

import "github.com/naskung/api-Costbook/internal/utils"

// Invoice เอกสารขาย 1 ใบ (Flutter เอาไปแสดงและสร้าง PDF)
// doc_type: tax_invoice = ใบกำกับภาษี (ร้านจด VAT), receipt = ใบเสร็จรับเงิน (ร้านไม่จด VAT ไม่มี VAT)
// ข้อมูลผู้ขาย / ผู้ซื้อ / ชื่อสินค้า เป็นค่า ณ วันที่ออก แก้ข้อมูลร้านหรือสินค้าทีหลัง ใบนี้ไม่เปลี่ยน
// status: issued = ออกแล้ว, cancelled = ยกเลิกแล้ว
// เงิน: items_total = รวมราคาสินค้า, discount = ส่วนลดทั้งบิล, subtotal = มูลค่าก่อน VAT, vat, total = ยอดสุทธิ
// prices_include_vat = true → ราคาในบรรทัดเป็นราคารวม VAT แล้ว
type Invoice struct {
	ID               string        `json:"id"`
	DocType          string        `json:"doc_type"`
	InvoiceNo        string        `json:"invoice_no"`
	IssueDate        string        `json:"issue_date"`
	Status           string        `json:"status"`
	SellerName       string        `json:"seller_name"`
	SellerAddress    string        `json:"seller_address"`
	SellerTaxID      string        `json:"seller_tax_id"`
	SellerBranch     string        `json:"seller_branch"`
	SellerPhone      string        `json:"seller_phone"`
	CustomerName     string        `json:"customer_name"`
	CustomerAddress  string        `json:"customer_address"`
	CustomerTaxID    string        `json:"customer_tax_id"`
	CustomerBranch   string        `json:"customer_branch"`
	Note             string        `json:"note"`
	PricesIncludeVAT bool          `json:"prices_include_vat"`
	ItemsTotal       float64       `json:"items_total"`
	Discount         float64       `json:"discount"`
	Subtotal         float64       `json:"subtotal"`
	VAT              float64       `json:"vat"`
	Total            float64       `json:"total"`
	Items            []InvoiceItem `json:"items"`
}

// InvoiceItem สินค้า 1 บรรทัดในเอกสาร (amount = quantity × unit_price)
type InvoiceItem struct {
	ID        string  `json:"id"`
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Unit      string  `json:"unit"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Amount    float64 `json:"amount"`
}

// InvoiceSummary ข้อมูลย่อของเอกสาร ใช้ในหน้ารายการ
type InvoiceSummary struct {
	ID           string  `json:"id"`
	DocType      string  `json:"doc_type"`
	InvoiceNo    string  `json:"invoice_no"`
	IssueDate    string  `json:"issue_date"`
	Status       string  `json:"status"`
	CustomerName string  `json:"customer_name"`
	Total        float64 `json:"total"`
}

// InvoiceRequest ข้อมูลที่รับมาตอนออกเอกสาร
// ชนิดเอกสาร / ราคารวม VAT ไหม API เลือกเองจากรูปแบบร้าน (ตั้งค่าร้าน) ไม่ต้องส่งมา
// customer_name ต้องมีเฉพาะใบกำกับภาษี (ใบเสร็จว่างได้)
type InvoiceRequest struct {
	IssueDate       string        `json:"issue_date"`
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

// Validate เช็คข้อมูลก่อนออกเอกสาร คืน error code ("" = ผ่าน)
func (req InvoiceRequest) Validate() string {
	if !utils.IsDate(req.IssueDate) {
		return "invoice_date_invalid"
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
	if req.Discount < 0 || req.Discount > itemsTotal(req.Items) {
		return "invoice_discount_invalid"
	}
	return ""
}
