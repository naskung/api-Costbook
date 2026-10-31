package cost

import "strings"

// CostSheet ใบต้นทุนของสินค้า 1 ตัว: รายการต้นทุนทั้งหมด + จำนวนที่ผลิตได้ต่อรอบ + ต้นทุน / ราคาขายต่อชิ้น
type CostSheet struct {
	ProductID     string     `json:"product_id"`
	ProductName   string     `json:"product_name"`
	Unit          string     `json:"unit"`
	BatchQuantity int        `json:"batch_quantity"`
	CostPrice     float64    `json:"cost_price"`
	SellPrice     float64    `json:"sell_price"`
	Items         []CostItem `json:"items"`
}

// CostItem ต้นทุน 1 บรรทัด
// type: material = วัตถุดิบ, labor = ค่าแรง, overhead = ค่าใช้จ่ายอื่น
type CostItem struct {
	ID     string  `json:"id"`
	Type   string  `json:"type"`
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

// CostSheetRequest ข้อมูลที่รับมาตอนบันทึกหน้าคำนวณต้นทุน (ส่งมาทั้งใบ รายการเดิมจะถูกแทนที่ทั้งหมด)
type CostSheetRequest struct {
	BatchQuantity int               `json:"batch_quantity"`
	SellPrice     float64           `json:"sell_price"`
	Items         []CostItemRequest `json:"items"`
}

type CostItemRequest struct {
	Type   string  `json:"type"`
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

// Validate เช็คข้อมูลก่อนบันทึก คืน error code ("" = ผ่าน)
func (req CostSheetRequest) Validate() string {
	if req.BatchQuantity <= 0 {
		return "cost_batch_quantity_invalid"
	}
	if req.SellPrice < 0 {
		return "product_price_invalid"
	}
	for _, item := range req.Items {
		code := item.validate()
		if code != "" {
			return code
		}
	}
	return ""
}

func (item CostItemRequest) validate() string {
	if item.Type != "material" && item.Type != "labor" && item.Type != "overhead" {
		return "cost_type_invalid"
	}
	if strings.TrimSpace(item.Name) == "" {
		return "cost_name_required"
	}
	if item.Amount < 0 {
		return "cost_amount_invalid"
	}
	return ""
}
