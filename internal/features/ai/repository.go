package ai

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/naskung/api-Costbook/internal/features/product"
)

// ErrNotFound ใช้บอก handler ว่าสินค้านี้ยังไม่เคยเช็คราคาตลาด (handler จะตอบ 404)
var ErrNotFound = errors.New("market check not found")

// Repository คือรายการคำสั่งที่คุยกับ DB (ตาราง market_checks + key ของ AI ในตาราง company)
// key ที่เก็บ / อ่านเป็นแบบเข้ารหัสแล้วเสมอ (ว่าง = ไม่มี)
type Repository interface {
	LatestMarketCheck(ctx context.Context, productID string) (MarketCheck, error)
	SaveMarketCheck(ctx context.Context, p product.Product, report MarketReport) (MarketCheck, error)
	SavedClaudeKey(ctx context.Context) (string, error)
	SaveClaudeKey(ctx context.Context, encrypted string) error
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const marketColumns = `id, product_id, our_cost_price, our_sell_price,
	retail_low, retail_high, retail_typical, wholesale_low, wholesale_high, wholesale_note,
	market_cost, suggested_price, confidence, summary, advice, cost_items, sources, created_at`

// LatestMarketCheck ผลเช็คครั้งล่าสุดของสินค้า (ไม่เคยเช็ค → ErrNotFound)
func (r *PostgresRepository) LatestMarketCheck(ctx context.Context, productID string) (MarketCheck, error) {
	row := r.db.QueryRow(ctx, `
		SELECT `+marketColumns+`
		FROM market_checks
		WHERE product_id = $1
		ORDER BY created_at DESC
		LIMIT 1`, productID)
	m, err := scanMarketCheck(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return MarketCheck{}, ErrNotFound
	}
	return m, err
}

// SaveMarketCheck บันทึกผลเช็คใหม่ + เก็บต้นทุน / ราคาขายของร้าน ณ ตอนนี้ไว้เทียบ
// cost_items / sources เป็น JSONB → pgx แปลง slice ของ Go เป็น JSON ให้เอง
func (r *PostgresRepository) SaveMarketCheck(ctx context.Context, p product.Product, report MarketReport) (MarketCheck, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO market_checks (product_id, our_cost_price, our_sell_price,
			retail_low, retail_high, retail_typical, wholesale_low, wholesale_high, wholesale_note,
			market_cost, suggested_price, confidence, summary, advice, cost_items, sources)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING `+marketColumns,
		p.ID, p.CostPrice, p.SellPrice,
		report.RetailLow, report.RetailHigh, report.RetailTypical, report.WholesaleLow, report.WholesaleHigh, report.WholesaleNote,
		report.MarketCost, report.SuggestedPrice, report.Confidence, report.Summary, report.Advice, report.CostItems, report.Sources)
	return scanMarketCheck(row)
}

// SavedClaudeKey key ของ Claude ที่ร้านใส่ไว้ (เข้ารหัสอยู่)
func (r *PostgresRepository) SavedClaudeKey(ctx context.Context) (string, error) {
	var encrypted string
	err := r.db.QueryRow(ctx, `SELECT ai_key FROM company WHERE id = 1`).Scan(&encrypted)
	return encrypted, err
}

// SaveClaudeKey บันทึก key ที่เข้ารหัสแล้ว (ส่ง "" = ลบ key กลับไปใช้ AI ฟรี)
func (r *PostgresRepository) SaveClaudeKey(ctx context.Context, encrypted string) error {
	_, err := r.db.Exec(ctx, `UPDATE company SET ai_key = $1, updated_at = now() WHERE id = 1`, encrypted)
	return err
}

// scanMarketCheck อ่าน 1 แถวของ market_checks (ลำดับตาม marketColumns)
func scanMarketCheck(row pgx.Row) (MarketCheck, error) {
	var m MarketCheck
	err := row.Scan(&m.ID, &m.ProductID, &m.OurCostPrice, &m.OurSellPrice,
		&m.RetailLow, &m.RetailHigh, &m.RetailTypical, &m.WholesaleLow, &m.WholesaleHigh, &m.WholesaleNote,
		&m.MarketCost, &m.SuggestedPrice, &m.Confidence, &m.Summary, &m.Advice, &m.CostItems, &m.Sources, &m.CheckedAt)
	return m, err
}
