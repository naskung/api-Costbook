package product

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound ใช้บอก handler ว่าไม่เจอสินค้า (handler จะตอบ 404)
var ErrNotFound = errors.New("product not found")

// ErrInUse สินค้าเคยอยู่ในใบกำกับแล้ว ลบไม่ได้ (handler จะตอบ 409)
var ErrInUse = errors.New("product is used in an invoice")

// Repository คือรายการคำสั่งที่คุยกับ DB
// ทำเป็น interface เพื่อให้ตอน test ใช้ fakeRepository แทน DB จริงได้
type Repository interface {
	List(ctx context.Context) ([]Product, error)
	FindByID(ctx context.Context, id string) (Product, error)
	Create(ctx context.Context, req ProductRequest) (Product, error)
	Update(ctx context.Context, id string, req ProductRequest) (Product, error)
	Delete(ctx context.Context, id string) error
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// productColumns column ที่ใช้อ่านสินค้า เรียงตรงกับ scanProduct
const productColumns = "id, name, unit, cost_price, sell_price, stock, min_stock, batch_quantity"

func (r *PostgresRepository) List(ctx context.Context) ([]Product, error) {
	rows, err := r.db.Query(ctx, `SELECT `+productColumns+` FROM products ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []Product{} // ไม่ใช้ nil → JSON จะได้ [] แทน null
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (r *PostgresRepository) FindByID(ctx context.Context, id string) (Product, error) {
	row := r.db.QueryRow(ctx, `SELECT `+productColumns+` FROM products WHERE id = $1`, id)
	return scanProduct(row)
}

func (r *PostgresRepository) Create(ctx context.Context, req ProductRequest) (Product, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO products (name, unit, cost_price, sell_price, min_stock)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+productColumns,
		req.Name, req.Unit, req.CostPrice, req.SellPrice, req.MinStock)
	return scanProduct(row)
}

func (r *PostgresRepository) Update(ctx context.Context, id string, req ProductRequest) (Product, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE products
		SET name = $2, unit = $3, cost_price = $4, sell_price = $5, min_stock = $6, updated_at = now()
		WHERE id = $1
		RETURNING `+productColumns,
		id, req.Name, req.Unit, req.CostPrice, req.SellPrice, req.MinStock)
	return scanProduct(row)
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	var inUse bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM invoice_items WHERE product_id = $1)`, id).Scan(&inUse)
	if err != nil {
		return err
	}
	if inUse {
		return ErrInUse
	}

	result, err := r.db.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// scanProduct อ่านผลจาก SELECT / RETURNING ใส่ Product
func scanProduct(row pgx.Row) (Product, error) {
	var p Product
	err := row.Scan(&p.ID, &p.Name, &p.Unit, &p.CostPrice, &p.SellPrice, &p.Stock, &p.MinStock, &p.BatchQuantity)
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	return p, err
}
