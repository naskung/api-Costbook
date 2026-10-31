package cost

import "strings"

// percentMax % สูงสุดของบรรทัดแบบเปอร์เซ็นต์ กันพิมพ์ผิดจนต้นทุนเพี้ยน
const percentMax = 1000

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
// kind: fixed = quantity × unit_price (เช่น 40 กรัม × 0.45), percent = percent% ของรวมบรรทัด fixed (เช่น เผื่อเสีย 10%)
// amount = จำนวนเงินที่ API คำนวณให้
type CostItem struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"`
	Kind      string  `json:"kind"`
	Name      string  `json:"name"`
	Quantity  float64 `json:"quantity"`
	Unit      string  `json:"unit"`
	UnitPrice float64 `json:"unit_price"`
	Percent   float64 `json:"percent"`
	Amount    float64 `json:"amount"`
}

// CostSheetRequest ข้อมูลที่รับมาตอนบันทึกหน้าคำนวณต้นทุน (ส่งมาทั้งใบ รายการเดิมจะถูกแทนที่ทั้งหมด)
type CostSheetRequest struct {
	BatchQuantity int               `json:"batch_quantity"`
	SellPrice     float64           `json:"sell_price"`
	Items         []CostItemRequest `json:"items"`
}

type CostItemRequest struct {
	Type      string  `json:"type"`
	Kind      string  `json:"kind"`
	Name      string  `json:"name"`
	Quantity  float64 `json:"quantity"`
	Unit      string  `json:"unit"`
	UnitPrice float64 `json:"unit_price"`
	Percent   float64 `json:"percent"`
}

// CostPreset รายการต้นทุนที่ใช้บ่อย (เช่น เส้น PLA กรัมละ 0.45, ค่าเครื่องชั่วโมงละ 6) ไว้เลือกซ้ำในหน้าคำนวณต้นทุน
type CostPreset struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"`
	Kind      string  `json:"kind"`
	Name      string  `json:"name"`
	Unit      string  `json:"unit"`
	UnitPrice float64 `json:"unit_price"`
	Percent   float64 `json:"percent"`
}

type PresetRequest struct {
	Type      string  `json:"type"`
	Kind      string  `json:"kind"`
	Name      string  `json:"name"`
	Unit      string  `json:"unit"`
	UnitPrice float64 `json:"unit_price"`
	Percent   float64 `json:"percent"`
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
	if item.Kind != "fixed" && item.Kind != "percent" {
		return "cost_kind_invalid"
	}
	if strings.TrimSpace(item.Name) == "" {
		return "cost_name_required"
	}
	if item.Quantity < 0 || item.UnitPrice < 0 {
		return "cost_amount_invalid"
	}
	if item.Percent < 0 || item.Percent > percentMax {
		return "cost_percent_invalid"
	}
	return ""
}

// Validate รายการที่ใช้บ่อยใช้กฎเดียวกับบรรทัดต้นทุน
func (req PresetRequest) Validate() string {
	item := CostItemRequest{
		Type: req.Type, Kind: req.Kind, Name: req.Name, Quantity: 1, Unit: req.Unit,
		UnitPrice: req.UnitPrice, Percent: req.Percent,
	}
	return item.validate()
}
