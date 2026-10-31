package stock

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrProductNotFound ใช้บอก handler ว่าไม่เจอสินค้า (handler จะตอบ 404)
var ErrProductNotFound = errors.New("product not found")

// ErrNotEnough เบิกออกมากกว่าที่มีอยู่ (handler จะตอบ 409)
var ErrNotEnough = errors.New("stock not enough")

// Repository คือรายการคำสั่งที่คุยกับ DB
// ทำเป็น interface เพื่อให้ตอน test ใช้ fakeRepository แทน DB จริงได้
type Repository interface {
	List(ctx context.Context, onlyLow bool) ([]StockItem, error)
	Movements(ctx context.Context, productID string) ([]Movement, error)
	AddMovement(ctx context.Context, movementType string, req MovementRequest) (Movement, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// List จำนวนคงเหลือของทุกสินค้า (onlyLow = true → เฉพาะที่ใกล้หมด)
func (r *PostgresRepository) List(ctx context.Context, onlyLow bool) ([]StockItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, unit, stock, min_stock, stock <= min_stock
		FROM products
		WHERE $1 = false OR stock <= min_stock
		ORDER BY stock <= min_stock DESC, name`, onlyLow)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []StockItem{} // ไม่ใช้ nil → JSON จะได้ [] แทน null
	for rows.Next() {
		var item StockItem
		err := rows.Scan(&item.ProductID, &item.Name, &item.Unit, &item.Stock, &item.MinStock, &item.IsLow)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// movementSelect อ่านประวัติพร้อมชื่อสินค้าและเลขที่ใบกำกับ เรียงตรงกับ scanMovement
const movementSelect = `
	SELECT m.id, m.product_id, p.name, p.unit, m.type, m.quantity, m.note, COALESCE(i.invoice_no, ''), m.created_at
	FROM stock_movements m
	JOIN products p ON p.id = m.product_id
	LEFT JOIN invoices i ON i.id = m.invoice_id`

// Movements ประวัติล่าสุด 200 รายการ (productID ว่าง = ทุกสินค้า)
func (r *PostgresRepository) Movements(ctx context.Context, productID string) ([]Movement, error) {
	rows, err := r.db.Query(ctx, movementSelect+`
		WHERE $1 = '' OR m.product_id::text = $1
		ORDER BY m.created_at DESC, p.name
		LIMIT 200`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	movements := []Movement{}
	for rows.Next() {
		m, err := scanMovement(rows)
		if err != nil {
			return nil, err
		}
		movements = append(movements, m)
	}
	return movements, rows.Err()
}

// AddMovement รับเข้า (in) / เบิกออก (out): แก้จำนวนคงเหลือ + บันทึกประวัติ ทำใน transaction เดียว
func (r *PostgresRepository) AddMovement(ctx context.Context, movementType string, req MovementRequest) (Movement, error) {
	change := req.Quantity
	if movementType == "out" {
		change = -req.Quantity
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Movement{}, err
	}
	defer tx.Rollback(ctx) // ถ้า Commit แล้ว คำสั่งนี้ไม่ทำอะไร

	// stock + change >= 0 → เบิกเกินที่มีไม่ได้ (ไม่มีแถวไหนถูกแก้)
	result, err := tx.Exec(ctx, `
		UPDATE products SET stock = stock + $1, updated_at = now()
		WHERE id = $2 AND stock + $1 >= 0`, change, req.ProductID)
	if err != nil {
		return Movement{}, err
	}
	if result.RowsAffected() == 0 {
		return Movement{}, r.notFoundOrNotEnough(ctx, req.ProductID)
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO stock_movements (product_id, type, quantity, note)
		VALUES ($1, $2, $3, $4)
		RETURNING id`, req.ProductID, movementType, req.Quantity, req.Note).Scan(&id)
	if err != nil {
		return Movement{}, err
	}

	movement, err := scanMovement(tx.QueryRow(ctx, movementSelect+` WHERE m.id = $1`, id))
	if err != nil {
		return Movement{}, err
	}
	return movement, tx.Commit(ctx)
}

// notFoundOrNotEnough แก้จำนวนไม่ได้ เพราะไม่มีสินค้านี้ หรือของไม่พอ
func (r *PostgresRepository) notFoundOrNotEnough(ctx context.Context, productID string) error {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM products WHERE id = $1)`, productID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrProductNotFound
	}
	return ErrNotEnough
}

// scanMovement อ่านผลจาก movementSelect ใส่ Movement
func scanMovement(row pgx.Row) (Movement, error) {
	var m Movement
	err := row.Scan(&m.ID, &m.ProductID, &m.ProductName, &m.Unit, &m.Type, &m.Quantity, &m.Note, &m.InvoiceNo, &m.CreatedAt)
	return m, err
}
