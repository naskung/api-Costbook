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
	SavedAI(ctx context.Context) (SavedAI, error)
	SaveAI(ctx context.Context, saved SavedAI) error
}

// SavedAI AI ที่ร้านตั้งไว้ในหน้าตั้งค่า (ตาราง company) — Provider ว่าง = ยังไม่ได้ตั้ง
// Provider = claude / openai, BaseURL + Model ใช้เฉพาะ openai, EncryptedKey = key ที่เข้ารหัสแล้ว (ว่างได้ เช่น Ollama ในเครื่อง)
type SavedAI struct {
	Provider     string
	BaseURL      string
	Model        string
	EncryptedKey string
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const marketColumns = `id, product_id, our_cost_price, our_sell_price,
	retail_low, retail_high, retail_typical, wholesale_low, wholesale_high, wholesale_note,
	market_cost, suggested_price, confidence, summary, advice, cost_items, sources, search_suggestions, created_at`

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
			market_cost, suggested_price, confidence, summary, advice, cost_items, sources, search_suggestions)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING `+marketColumns,
		p.ID, p.CostPrice, p.SellPrice,
		report.RetailLow, report.RetailHigh, report.RetailTypical, report.WholesaleLow, report.WholesaleHigh, report.WholesaleNote,
		report.MarketCost, report.SuggestedPrice, report.Confidence, report.Summary, report.Advice, report.CostItems, report.Sources,
		report.SearchSuggestions)
	return scanMarketCheck(row)
}

// SavedAI AI ที่ร้านตั้งไว้ (key ยังเข้ารหัสอยู่)
func (r *PostgresRepository) SavedAI(ctx context.Context) (SavedAI, error) {
	var s SavedAI
	err := r.db.QueryRow(ctx, `SELECT ai_provider, ai_base_url, ai_model, ai_key FROM company WHERE id = 1`).
		Scan(&s.Provider, &s.BaseURL, &s.Model, &s.EncryptedKey)
	return s, err
}

// SaveAI บันทึก AI ที่ร้านเลือก (ส่ง SavedAI{} = ลบ กลับไปใช้ AI จาก .env)
func (r *PostgresRepository) SaveAI(ctx context.Context, s SavedAI) error {
	_, err := r.db.Exec(ctx, `
		UPDATE company SET ai_provider = $1, ai_base_url = $2, ai_model = $3, ai_key = $4, updated_at = now()
		WHERE id = 1`,
		s.Provider, s.BaseURL, s.Model, s.EncryptedKey)
	return err
}

// scanMarketCheck อ่าน 1 แถวของ market_checks (ลำดับตาม marketColumns)
func scanMarketCheck(row pgx.Row) (MarketCheck, error) {
	var m MarketCheck
	err := row.Scan(&m.ID, &m.ProductID, &m.OurCostPrice, &m.OurSellPrice,
		&m.RetailLow, &m.RetailHigh, &m.RetailTypical, &m.WholesaleLow, &m.WholesaleHigh, &m.WholesaleNote,
		&m.MarketCost, &m.SuggestedPrice, &m.Confidence, &m.Summary, &m.Advice, &m.CostItems, &m.Sources,
		&m.SearchSuggestions, &m.CheckedAt)
	return m, err
}
