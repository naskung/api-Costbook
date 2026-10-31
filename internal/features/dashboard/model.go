package dashboard

// Dashboard ตัวเลขสรุปของหน้าแรก (เดือนที่เลือก)
// invoice_count / invoice_total นับเฉพาะเอกสารขายที่ยังไม่ยกเลิก / low_stock ไม่นับสินค้าที่ไม่นับสต็อก
type Dashboard struct {
	Month         string         `json:"month"`
	Income        float64        `json:"income"`
	Expense       float64        `json:"expense"`
	Profit        float64        `json:"profit"`
	InvoiceCount  int            `json:"invoice_count"`
	InvoiceTotal  float64        `json:"invoice_total"`
	ProductCount  int            `json:"product_count"`
	LowStockCount int            `json:"low_stock_count"`
	LowStock      []LowStockItem `json:"low_stock"`
}

// LowStockItem สินค้าที่ใกล้หมด (คงเหลือ ≤ จำนวนขั้นต่ำ)
type LowStockItem struct {
	ProductID string `json:"product_id"`
	Name      string `json:"name"`
	Unit      string `json:"unit"`
	Stock     int    `json:"stock"`
	MinStock  int    `json:"min_stock"`
}
