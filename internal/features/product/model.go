package product

import "strings"

// Product ข้อมูลสินค้า (ชื่อ json ต้องตรงกับ ProductModel ฝั่ง Flutter)
// stock เปลี่ยนผ่านหน้าสต็อก / ใบกำกับเท่านั้น, batch_quantity แก้ในหน้าคำนวณต้นทุน
type Product struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Unit          string  `json:"unit"`
	CostPrice     float64 `json:"cost_price"`
	SellPrice     float64 `json:"sell_price"`
	Stock         int     `json:"stock"`
	MinStock      int     `json:"min_stock"`
	BatchQuantity int     `json:"batch_quantity"`
}

// ProductRequest ข้อมูลที่รับมาตอนเพิ่ม / แก้สินค้า
type ProductRequest struct {
	Name      string  `json:"name"`
	Unit      string  `json:"unit"`
	CostPrice float64 `json:"cost_price"`
	SellPrice float64 `json:"sell_price"`
	MinStock  int     `json:"min_stock"`
}

// Validate เช็คข้อมูลก่อนบันทึก คืน error code ("" = ผ่าน)
func (req ProductRequest) Validate() string {
	if strings.TrimSpace(req.Name) == "" {
		return "product_name_required"
	}
	if req.CostPrice < 0 || req.SellPrice < 0 {
		return "product_price_invalid"
	}
	if req.MinStock < 0 {
		return "product_min_stock_invalid"
	}
	return ""
}
