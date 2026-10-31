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
// type: in = รับเข้า, out = เบิกออก / invoice_no ไม่ว่าง = เกิดจากใบกำกับ (ออกใบ = out, ยกเลิกใบ = in)
type Movement struct {
	ID          string    `json:"id"`
	ProductID   string    `json:"product_id"`
	ProductName string    `json:"product_name"`
	Unit        string    `json:"unit"`
	Type        string    `json:"type"`
	Quantity    int       `json:"quantity"`
	Note        string    `json:"note"`
	InvoiceNo   string    `json:"invoice_no"`
	CreatedAt   time.Time `json:"created_at"`
}

// MovementRequest ข้อมูลที่รับมาตอนรับเข้า / เบิกออก
type MovementRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Note      string `json:"note"`
}

// Validate เช็คข้อมูลก่อนบันทึก คืน error code ("" = ผ่าน)
func (req MovementRequest) Validate() string {
	if !utils.IsUUID(req.ProductID) {
		return "stock_product_required"
	}
	if req.Quantity <= 0 {
		return "stock_quantity_invalid"
	}
	return ""
}
