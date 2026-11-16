package dashboard

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/naskung/api-Costbook/internal/features/transaction"
	"github.com/naskung/api-Costbook/internal/utils"
)

// lowStockLimit แสดงสินค้าใกล้หมดบนหน้าแรกไม่เกินกี่ตัว (ดูทั้งหมดได้ที่หน้าสต็อก)
const lowStockLimit = 5

// topProductLimit สินค้าขายดีกี่อันดับ / trendMonthCount กราฟย้อนหลังกี่เดือน (รวมเดือนของวันสุดท้ายที่เลือก)
const (
	topProductLimit = 5
	trendMonthCount = 6
)

// issuedInPeriod เงื่อนไขของเอกสารขายที่นับยอด: ยังไม่ยกเลิก และออกในช่วงที่เลือก ($1 = วันแรก, $2 = วันสุดท้าย นับรวม)
const issuedInPeriod = `i.status = 'issued' AND i.issue_date BETWEEN $1::date AND $2::date`

// netAmount ยอดของบรรทัดหลังหักส่วนลด และไม่รวม VAT
// = ยอดบรรทัด × (มูลค่าก่อน VAT ทั้งใบ ÷ ยอดรวมสินค้าทั้งใบ) → แบ่งส่วนลด / VAT ตามสัดส่วนยอดของแต่ละบรรทัด
const netAmount = `ii.amount * CASE WHEN i.items_total > 0 THEN i.subtotal / i.items_total ELSE 0 END`

// Repository คือรายการคำสั่งที่คุยกับ DB
// period = ช่วงวันที่ที่เลือก เช่น 2026-11-01 ถึง 2026-11-16 (นับรวมทั้ง 2 วัน)
type Repository interface {
	Get(ctx context.Context, period utils.DateRange) (Dashboard, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Get อ่านทุกตัวเลขของหน้าภาพรวม ทีละส่วน
// previous = ช่วงก่อนหน้าที่เอามาเทียบ (ทั้งเดือน → เดือนก่อน / นอกนั้น → ช่วงที่ยาวเท่ากันก่อนหน้า)
func (r *PostgresRepository) Get(ctx context.Context, period utils.DateRange) (Dashboard, error) {
	previous := utils.PreviousRange(period)
	d := Dashboard{From: period.From, To: period.To, PreviousFrom: previous.From, PreviousTo: previous.To}

	amounts, err := transaction.ScanAmounts(r.db.QueryRow(ctx, transaction.AmountsSQL, period.From, period.To))
	if err != nil {
		return Dashboard{}, err
	}
	summary := transaction.NewSummary(period, amounts)
	d.Income = summary.Income
	d.Expense = summary.Expense
	d.Profit = summary.Profit
	d.Revenue = summary.Revenue
	d.OtherIncome = summary.OtherIncome
	d.MaterialExpense = summary.MaterialExpense
	d.OperatingExpense = summary.OperatingExpense

	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(total), 0)
		FROM invoices i
		WHERE `+issuedInPeriod, period.From, period.To).
		Scan(&d.InvoiceCount, &d.InvoiceTotal)
	if err != nil {
		return Dashboard{}, err
	}

	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE track_stock AND stock <= min_stock)
		FROM products`).
		Scan(&d.ProductCount, &d.LowStockCount)
	if err != nil {
		return Dashboard{}, err
	}

	d.LowStock, err = r.lowStock(ctx)
	if err != nil {
		return Dashboard{}, err
	}
	d.Sales, err = r.sales(ctx, period, d.InvoiceCount)
	if err != nil {
		return Dashboard{}, err
	}
	d.MonthlyTrend, err = r.monthlyTrend(ctx, period)
	if err != nil {
		return Dashboard{}, err
	}
	d.Previous, err = r.totals(ctx, previous)
	if err != nil {
		return Dashboard{}, err
	}
	d.DailySales, err = r.dailySales(ctx, period)
	if err != nil {
		return Dashboard{}, err
	}
	d.PreviousDaily, err = r.dailySales(ctx, previous)
	if err != nil {
		return Dashboard{}, err
	}
	d.TopProducts, err = r.topProducts(ctx, period)
	if err != nil {
		return Dashboard{}, err
	}
	d.CategorySales, err = r.categorySales(ctx, period)
	if err != nil {
		return Dashboard{}, err
	}
	d.ExpenseByCategory, err = r.expenseByCategory(ctx, period)
	if err != nil {
		return Dashboard{}, err
	}
	d.Stock, err = r.stockValue(ctx)
	if err != nil {
		return Dashboard{}, err
	}
	d.Market, err = r.market(ctx)
	return d, err
}

// lowStock สินค้าใกล้หมด (เฉพาะที่นับสต็อก) เรียงจากเหลือน้อยที่สุด (ของหมดแล้วขึ้นก่อน)
func (r *PostgresRepository) lowStock(ctx context.Context) ([]LowStockItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, unit, stock, min_stock
		FROM products
		WHERE track_stock AND stock <= min_stock
		ORDER BY stock, name
		LIMIT $1`, lowStockLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []LowStockItem{} // ไม่ใช้ nil → JSON จะได้ [] แทน null
	for rows.Next() {
		var item LowStockItem
		err := rows.Scan(&item.ProductID, &item.Name, &item.Unit, &item.Stock, &item.MinStock)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// sales ยอดขาย / VAT / ส่วนลด / ต้นทุนของที่ขายไป / จำนวนชิ้น ของช่วงที่เลือก
func (r *PostgresRepository) sales(ctx context.Context, period utils.DateRange, bills int) (SalesSummary, error) {
	var s SalesSummary
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(subtotal), 0), COALESCE(SUM(vat), 0), COALESCE(SUM(discount), 0)
		FROM invoices i
		WHERE `+issuedInPeriod, period.From, period.To).
		Scan(&s.NetSales, &s.VAT, &s.Discount)
	if err != nil {
		return SalesSummary{}, err
	}

	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(ii.quantity * ii.cost_price), 0), COALESCE(SUM(ii.quantity), 0)
		FROM invoice_items ii
		JOIN invoices i ON i.id = ii.invoice_id
		WHERE `+issuedInPeriod, period.From, period.To).
		Scan(&s.Cost, &s.ItemsSold)
	if err != nil {
		return SalesSummary{}, err
	}
	return salesSummary(s, bills), nil
}

// totals ยอดรวม + จำนวนบิลของ 1 ช่วง
// ใช้ SQL ตัวเดียวกับหน้ารายรับ-รายจ่าย (transaction.AmountsSQL) → กำไรตรงกันทุกหน้า
func (r *PostgresRepository) totals(ctx context.Context, period utils.DateRange) (MonthTotals, error) {
	amounts, err := transaction.ScanAmounts(r.db.QueryRow(ctx, transaction.AmountsSQL, period.From, period.To))
	if err != nil {
		return MonthTotals{}, err
	}
	var bills int
	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM invoices i WHERE `+issuedInPeriod, period.From, period.To).Scan(&bills)
	if err != nil {
		return MonthTotals{}, err
	}
	return monthTotals(period, bills, amounts), nil
}

// monthlyTrend ยอดทีละเดือน ย้อนหลัง 6 เดือน (เรียงเก่า → ใหม่ เดือนสุดท้าย = เดือนของวันสุดท้ายที่เลือก)
func (r *PostgresRepository) monthlyTrend(ctx context.Context, period utils.DateRange) ([]MonthTotals, error) {
	months := []MonthTotals{}
	for _, month := range trendMonths(period.To, trendMonthCount) {
		totals, err := r.totals(ctx, month)
		if err != nil {
			return nil, err
		}
		months = append(months, totals)
	}
	return months, nil
}

// dailySales ยอดขายรายวันของช่วงที่เลือก (มีทุกวัน วันที่ไม่ขายได้ 0)
func (r *PostgresRepository) dailySales(ctx context.Context, period utils.DateRange) ([]DailySales, error) {
	rows, err := r.db.Query(ctx, `
		SELECT to_char(d, 'YYYY-MM-DD'), COALESCE(SUM(i.subtotal), 0), COUNT(i.id)
		FROM generate_series($1::date, $2::date, interval '1 day') AS d
		LEFT JOIN invoices i ON i.issue_date = d::date AND i.status = 'issued'
		GROUP BY d
		ORDER BY d`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	days := []DailySales{}
	for rows.Next() {
		var day DailySales
		err := rows.Scan(&day.Date, &day.NetSales, &day.Bills)
		if err != nil {
			return nil, err
		}
		day.NetSales = utils.Round2(day.NetSales)
		days = append(days, day)
	}
	return days, rows.Err()
}

// topProducts สินค้าขายดี 5 อันดับ (เรียงตามยอดขายก่อน VAT หลังหักส่วนลด)
func (r *PostgresRepository) topProducts(ctx context.Context, period utils.DateRange) ([]ProductSales, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ii.product_id, MAX(ii.name), MAX(ii.unit), SUM(ii.quantity),
			SUM(`+netAmount+`), SUM(ii.quantity * ii.cost_price)
		FROM invoice_items ii
		JOIN invoices i ON i.id = ii.invoice_id
		WHERE `+issuedInPeriod+`
		GROUP BY ii.product_id
		ORDER BY 5 DESC
		LIMIT $3`, period.From, period.To, topProductLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []ProductSales{}
	for rows.Next() {
		var p ProductSales
		err := rows.Scan(&p.ProductID, &p.Name, &p.Unit, &p.Quantity, &p.NetSales, &p.Cost)
		if err != nil {
			return nil, err
		}
		products = append(products, productSales(p))
	}
	return products, rows.Err()
}

// categorySales ยอดขายตามหมวดสินค้า (หมวดปัจจุบันของสินค้า) มากไปน้อย
func (r *PostgresRepository) categorySales(ctx context.Context, period utils.DateRange) ([]CategoryAmount, error) {
	return r.categoryAmounts(ctx, `
		SELECT COALESCE(c.name, ''), SUM(`+netAmount+`)
		FROM invoice_items ii
		JOIN invoices i ON i.id = ii.invoice_id
		JOIN products p ON p.id = ii.product_id
		LEFT JOIN product_categories c ON c.id = p.category_id
		WHERE `+issuedInPeriod+`
		GROUP BY COALESCE(c.name, '')
		ORDER BY 2 DESC`, period)
}

// expenseByCategory รายจ่ายตามหมวด มากไปน้อย
func (r *PostgresRepository) expenseByCategory(ctx context.Context, period utils.DateRange) ([]CategoryAmount, error) {
	return r.categoryAmounts(ctx, `
		SELECT category, SUM(amount)
		FROM transactions
		WHERE type = 'expense' AND date BETWEEN $1::date AND $2::date
		GROUP BY category
		ORDER BY 2 DESC`, period)
}

// categoryAmounts อ่านผลแบบ (หมวด, ยอด) แล้วใส่ % ของแต่ละหมวด ($1 / $2 = วันแรก / วันสุดท้ายของช่วง)
func (r *PostgresRepository) categoryAmounts(ctx context.Context, sql string, period utils.DateRange) ([]CategoryAmount, error) {
	rows, err := r.db.Query(ctx, sql, period.From, period.To)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []CategoryAmount{}
	for rows.Next() {
		var item CategoryAmount
		err := rows.Scan(&item.Category, &item.Amount)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return withPercent(items), nil
}

// stockValue มูลค่าของในสต็อก (ทุน / ราคาขาย) + จำนวนสินค้าที่หมดแล้ว — เฉพาะที่นับสต็อก
func (r *PostgresRepository) stockValue(ctx context.Context) (StockValue, error) {
	var s StockValue
	err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(stock * cost_price), 0),
			COALESCE(SUM(stock * sell_price), 0),
			COUNT(*) FILTER (WHERE stock <= 0)
		FROM products
		WHERE track_stock`).
		Scan(&s.CostValue, &s.RetailValue, &s.OutOfStock)
	s.CostValue = utils.Round2(s.CostValue)
	s.RetailValue = utils.Round2(s.RetailValue)
	return s, err
}

// market ราคาของร้านเทียบกับผลเช็คราคาตลาดครั้งล่าสุดของแต่ละสินค้า
func (r *PostgresRepository) market(ctx context.Context) ([]MarketPosition, error) {
	var vatRegistered, pricesIncludeVAT bool
	err := r.db.QueryRow(ctx, `SELECT vat_registered, prices_include_vat FROM company WHERE id = 1`).
		Scan(&vatRegistered, &pricesIncludeVAT)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT ON (m.product_id)
			m.product_id, p.name, p.sell_price, m.retail_low, m.retail_high, m.retail_typical, m.created_at
		FROM market_checks m
		JOIN products p ON p.id = m.product_id
		ORDER BY m.product_id, m.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []MarketPosition{}
	for rows.Next() {
		var m MarketPosition
		var sellPrice float64
		err := rows.Scan(&m.ProductID, &m.Name, &sellPrice, &m.RetailLow, &m.RetailHigh, &m.RetailTypical, &m.CheckedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, marketPositionOf(m, sellPrice, vatRegistered, pricesIncludeVAT))
	}
	return items, rows.Err()
}
