package stock

import (
	"time"

	"github.com/naskung/api-Costbook/internal/utils"
)

// StockItem จำนวนคงเหลือของสินค้า 1 ตัว
// is_low = true เมื่อคงเหลือ ≤ จำนวนขั้นต่ำ (ใกล้หมด)
type StockItem struct {
	ProductID string `json:"product_id"`
	Name      string `json:"name"`
	Unit      string `json:"unit"`
	Stock     int    `json:"stock"`
	MinStock  int    `json:"min_stock"`
	IsLow     bool   `json:"is_low"`
}

// Movement ประวัติรับเข้า / เบิกออก 1 ครั้ง
// type: in = รับเข้า, out = เบิกออก / invoice_no ไม่ว่าง = เกิดจากเอกสารขาย (ออกใบ = out, ยกเลิกใบ = in)
// unit_cost = ราคาทุนต่อหน่วยตอนรับเข้า (0 = ไม่ได้ใส่), supplier = ซื้อจากใคร
type Movement struct {
	ID          string    `json:"id"`
	ProductID   string    `json:"product_id"`
	ProductName string    `json:"product_name"`
	Unit        string    `json:"unit"`
	Type        string    `json:"type"`
	Quantity    int       `json:"quantity"`
	Note        string    `json:"note"`
	InvoiceNo   string    `json:"invoice_no"`
	UnitCost    float64   `json:"unit_cost"`
	Supplier    string    `json:"supplier"`
	CreatedAt   time.Time `json:"created_at"`
}

// MovementRequest ข้อมูลที่รับมาตอนรับเข้า / เบิกออก
// unit_cost (ใช้ตอนรับเข้า ไม่ใส่ก็ได้) > 0 → คิดต้นทุนเฉลี่ยของสินค้าใหม่
type MovementRequest struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Note      string  `json:"note"`
	UnitCost  float64 `json:"unit_cost"`
	Supplier  string  `json:"supplier"`
}

// Validate เช็คข้อมูลก่อนบันทึก คืน error code ("" = ผ่าน)
func (req MovementRequest) Validate() string {
	if !utils.IsUUID(req.ProductID) {
		return "stock_product_required"
	}
	if req.Quantity <= 0 {
		return "stock_quantity_invalid"
	}
	if req.UnitCost < 0 {
		return "stock_unit_cost_invalid"
	}
	return ""
}
