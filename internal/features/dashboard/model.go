package dashboard

import "time"

// Dashboard ตัวเลขสรุปของหน้าแรก (เดือนที่เลือก)
// invoice_count / invoice_total นับเฉพาะเอกสารขายที่ยังไม่ยกเลิก / low_stock ไม่นับสินค้าที่ไม่นับสต็อก
// income / expense = เงินเข้า / ออกตามที่บันทึก, revenue = รายได้ (ยอดขายก่อน VAT + รายรับอื่น)
// profit = กำไรสุทธิ = revenue − ต้นทุนสินค้าที่ขาย (sales.cost) − operating_expense (รายจ่ายไม่รวมซื้อวัตถุดิบ)
// คิดด้วย transaction.NewSummary ตัวเดียวกับหน้ารายรับ-รายจ่าย
type Dashboard struct {
	Month            string  `json:"month"`
	Income           float64 `json:"income"`
	Expense          float64 `json:"expense"`
	Profit           float64 `json:"profit"`
	Revenue          float64 `json:"revenue"`
	OtherIncome      float64 `json:"other_income"`
	MaterialExpense  float64 `json:"material_expense"`
	OperatingExpense float64 `json:"operating_expense"`

	InvoiceCount  int            `json:"invoice_count"`
	InvoiceTotal  float64        `json:"invoice_total"`
	ProductCount  int            `json:"product_count"`
	LowStockCount int            `json:"low_stock_count"`
	LowStock      []LowStockItem `json:"low_stock"`

	Sales             SalesSummary     `json:"sales"`
	Previous          MonthTotals      `json:"previous"`
	MonthlyTrend      []MonthTotals    `json:"monthly_trend"`
	DailySales        []DailySales     `json:"daily_sales"`
	PreviousDaily     []DailySales     `json:"previous_daily_sales"`
	TopProducts       []ProductSales   `json:"top_products"`
	CategorySales     []CategoryAmount `json:"category_sales"`
	ExpenseByCategory []CategoryAmount `json:"expense_by_category"`
	Stock             StockValue       `json:"stock"`
	Market            []MarketPosition `json:"market"`
}

// LowStockItem สินค้าที่ใกล้หมด (คงเหลือ ≤ จำนวนขั้นต่ำ)
type LowStockItem struct {
	ProductID string `json:"product_id"`
	Name      string `json:"name"`
	Unit      string `json:"unit"`
	Stock     int    `json:"stock"`
	MinStock  int    `json:"min_stock"`
}

// SalesSummary ยอดขายจากเอกสารขายของเดือน
// net_sales = ยอดขายก่อน VAT หลังหักส่วนลด (เงินที่ร้านได้จริง) / cost = ต้นทุนของที่ขายไป (ต้นทุน ณ วันที่ขาย)
// gross_margin = กำไรขั้นต้น ÷ ยอดขาย × 100
type SalesSummary struct {
	NetSales    float64 `json:"net_sales"`
	Cost        float64 `json:"cost"`
	GrossProfit float64 `json:"gross_profit"`
	GrossMargin float64 `json:"gross_margin"`
	VAT         float64 `json:"vat"`
	Discount    float64 `json:"discount"`
	ItemsSold   int     `json:"items_sold"`
	AverageBill float64 `json:"average_bill"`
}

// MonthTotals ยอดของ 1 เดือน (ไว้ทำกราฟย้อนหลัง และเทียบกับเดือนก่อน) — ความหมายเหมือน Dashboard / transaction.Summary
// net_sales / bills / gross_profit / cost_of_sales มาจากเอกสารขาย, profit = กำไรสุทธิ
type MonthTotals struct {
	Month            string  `json:"month"`
	Income           float64 `json:"income"`
	Expense          float64 `json:"expense"`
	Profit           float64 `json:"profit"`
	NetSales         float64 `json:"net_sales"`
	Bills            int     `json:"bills"`
	GrossProfit      float64 `json:"gross_profit"`
	Revenue          float64 `json:"revenue"`
	CostOfSales      float64 `json:"cost_of_sales"`
	OperatingExpense float64 `json:"operating_expense"`
}

// DailySales ยอดขายรายวัน (มีทุกวันของเดือน วันที่ไม่มีขายได้ 0)
type DailySales struct {
	Date     string  `json:"date"`
	NetSales float64 `json:"net_sales"`
	Bills    int     `json:"bills"`
}

// ProductSales สินค้าขายดี: ยอดขาย (ก่อน VAT หลังลด) / ต้นทุน / กำไร / margin %
type ProductSales struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Unit      string  `json:"unit"`
	Quantity  int     `json:"quantity"`
	NetSales  float64 `json:"net_sales"`
	Cost      float64 `json:"cost"`
	Profit    float64 `json:"profit"`
	Margin    float64 `json:"margin"`
}

// CategoryAmount ยอดตามหมวด + คิดเป็นกี่ % ของทั้งหมด (category ว่าง = ไม่มีหมวด)
type CategoryAmount struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
	Percent  float64 `json:"percent"`
}

// StockValue มูลค่าของในสต็อก (เฉพาะสินค้าที่นับสต็อก)
// cost_value = คงเหลือ × ต้นทุน, retail_value = คงเหลือ × ราคาขาย
type StockValue struct {
	CostValue   float64 `json:"cost_value"`
	RetailValue float64 `json:"retail_value"`
	OutOfStock  int     `json:"out_of_stock"`
}

// MarketPosition ราคาของร้านเทียบกับราคาตลาด (จากการเช็คด้วย AI ครั้งล่าสุด)
// customer_price = ราคาที่ลูกค้าจ่ายจริง (ร้านจด VAT ที่ตั้งราคาไม่รวม VAT → บวก VAT ให้ก่อนเทียบ)
// position: below = ถูกกว่าตลาด / within = อยู่ในช่วง / above = แพงกว่า / unknown = ไม่มีราคาตลาด
// diff_percent = ต่างจากราคาที่เจอบ่อยกี่ % (บวก = แพงกว่า)
type MarketPosition struct {
	ProductID     string    `json:"product_id"`
	Name          string    `json:"name"`
	CustomerPrice float64   `json:"customer_price"`
	RetailLow     float64   `json:"retail_low"`
	RetailHigh    float64   `json:"retail_high"`
	RetailTypical float64   `json:"retail_typical"`
	Position      string    `json:"position"`
	DiffPercent   float64   `json:"diff_percent"`
	CheckedAt     time.Time `json:"checked_at"`
}
