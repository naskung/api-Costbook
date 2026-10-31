package cost

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrProductNotFound ใช้บอก handler ว่าไม่เจอสินค้า (handler จะตอบ 404)
var ErrProductNotFound = errors.New("product not found")

// Repository คือรายการคำสั่งที่คุยกับ DB
// ทำเป็น interface เพื่อให้ตอน test ใช้ fakeRepository แทน DB จริงได้
type Repository interface {
	Get(ctx context.Context, productID string) (CostSheet, error)
	Save(ctx context.Context, productID string, req CostSheetRequest, costPrice float64) (CostSheet, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Get(ctx context.Context, productID string) (CostSheet, error) {
	var sheet CostSheet
	err := r.db.QueryRow(ctx, `
		SELECT id, name, unit, batch_quantity, cost_price, sell_price
		FROM products
		WHERE id = $1`, productID).
		Scan(&sheet.ProductID, &sheet.ProductName, &sheet.Unit, &sheet.BatchQuantity, &sheet.CostPrice, &sheet.SellPrice)
	if errors.Is(err, pgx.ErrNoRows) {
		return CostSheet{}, ErrProductNotFound
	}
	if err != nil {
		return CostSheet{}, err
	}

	sheet.Items, err = r.items(ctx, productID)
	return sheet, err
}

// Save แทนที่รายการต้นทุนทั้งหมดของสินค้า + บันทึกต้นทุน / ราคาขายต่อชิ้นใหม่ ทำใน transaction เดียว
func (r *PostgresRepository) Save(ctx context.Context, productID string, req CostSheetRequest, costPrice float64) (CostSheet, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return CostSheet{}, err
	}
	defer tx.Rollback(ctx) // ถ้า Commit แล้ว คำสั่งนี้ไม่ทำอะไร

	result, err := tx.Exec(ctx, `
		UPDATE products
		SET batch_quantity = $2, cost_price = $3, sell_price = $4, updated_at = now()
		WHERE id = $1`,
		productID, req.BatchQuantity, costPrice, req.SellPrice)
	if err != nil {
		return CostSheet{}, err
	}
	if result.RowsAffected() == 0 {
		return CostSheet{}, ErrProductNotFound
	}

	_, err = tx.Exec(ctx, `DELETE FROM product_costs WHERE product_id = $1`, productID)
	if err != nil {
		return CostSheet{}, err
	}
	for i, item := range req.Items {
		_, err = tx.Exec(ctx, `
			INSERT INTO product_costs (product_id, line_no, type, name, amount)
			VALUES ($1, $2, $3, $4, $5)`,
			productID, i+1, item.Type, item.Name, item.Amount)
		if err != nil {
			return CostSheet{}, err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return CostSheet{}, err
	}
	return r.Get(ctx, productID)
}

// items รายการต้นทุนของสินค้า เรียงตามประเภท แล้วตามลำดับที่กรอก
func (r *PostgresRepository) items(ctx context.Context, productID string) ([]CostItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, type, name, amount
		FROM product_costs
		WHERE product_id = $1
		ORDER BY CASE type WHEN 'material' THEN 1 WHEN 'labor' THEN 2 ELSE 3 END, line_no`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []CostItem{} // ไม่ใช้ nil → JSON จะได้ [] แทน null
	for rows.Next() {
		var item CostItem
		err := rows.Scan(&item.ID, &item.Type, &item.Name, &item.Amount)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
