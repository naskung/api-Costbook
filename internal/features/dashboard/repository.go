package dashboard

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/naskung/api-Costbook/internal/utils"
)

// lowStockLimit แสดงสินค้าใกล้หมดบนหน้าแรกไม่เกินกี่ตัว (ดูทั้งหมดได้ที่หน้าสต็อก)
const lowStockLimit = 5

// Repository คือรายการคำสั่งที่คุยกับ DB
// monthStart = วันแรกของเดือน เช่น "2026-10-01"
type Repository interface {
	Get(ctx context.Context, monthStart string) (Dashboard, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Get(ctx context.Context, monthStart string) (Dashboard, error) {
	d := Dashboard{Month: monthStart[:7]} // "2026-10-01" → "2026-10"

	err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE type = 'income'), 0),
			COALESCE(SUM(amount) FILTER (WHERE type = 'expense'), 0)
		FROM transactions
		WHERE date >= $1::date AND date < $1::date + interval '1 month'`, monthStart).
		Scan(&d.Income, &d.Expense)
	if err != nil {
		return Dashboard{}, err
	}
	d.Profit = utils.Profit(d.Income, d.Expense)

	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(total), 0)
		FROM invoices
		WHERE status = 'issued' AND issue_date >= $1::date AND issue_date < $1::date + interval '1 month'`, monthStart).
		Scan(&d.InvoiceCount, &d.InvoiceTotal)
	if err != nil {
		return Dashboard{}, err
	}

	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE stock <= min_stock)
		FROM products`).
		Scan(&d.ProductCount, &d.LowStockCount)
	if err != nil {
		return Dashboard{}, err
	}

	d.LowStock, err = r.lowStock(ctx)
	return d, err
}

// lowStock สินค้าใกล้หมด เรียงจากเหลือน้อยที่สุด (ของหมดแล้วขึ้นก่อน)
func (r *PostgresRepository) lowStock(ctx context.Context) ([]LowStockItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, unit, stock, min_stock
		FROM products
		WHERE stock <= min_stock
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
