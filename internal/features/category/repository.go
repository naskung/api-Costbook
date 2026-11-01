package category

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound ไม่เจอหมวดนี้ (handler จะตอบ 404)
var ErrNotFound = errors.New("product category not found")

// ErrNameTaken มีหมวดชื่อนี้แล้ว (handler จะตอบ 409)
var ErrNameTaken = errors.New("product category name is already used")

// uniqueViolation code ของ Postgres เมื่อบันทึกค่าที่ห้ามซ้ำแล้วซ้ำ
const uniqueViolation = "23505"

// Repository คือรายการคำสั่งที่คุยกับ DB (test ใช้ fakeRepository แทน)
type Repository interface {
	List(ctx context.Context) ([]Category, error)
	Create(ctx context.Context, req CategoryRequest) (Category, error)
	Update(ctx context.Context, id string, req CategoryRequest) (Category, error)
	Delete(ctx context.Context, id string) error
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// selectCategories หมวด + จำนวนสินค้าในหมวด (เรียงตามชื่อ)
const selectCategories = `
	SELECT c.id, c.name, COUNT(p.id)
	FROM product_categories c
	LEFT JOIN products p ON p.category_id = c.id`

func (r *PostgresRepository) List(ctx context.Context) ([]Category, error) {
	// เรียงตามรหัสตัวอักษร (COLLATE "C") — collation เริ่มต้นของบางเครื่องเรียงภาษาไทยเพี้ยน
	rows, err := r.db.Query(ctx, selectCategories+` GROUP BY c.id ORDER BY c.name COLLATE "C"`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []Category{} // ไม่ใช้ nil → JSON จะได้ [] แทน null
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.ProductCount); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (r *PostgresRepository) Create(ctx context.Context, req CategoryRequest) (Category, error) {
	var c Category
	err := r.db.QueryRow(ctx, `INSERT INTO product_categories (name) VALUES ($1) RETURNING id, name`, req.cleanName()).
		Scan(&c.ID, &c.Name)
	return c, nameTakenOr(err)
}

// Update แก้ชื่อหมวด → สินค้าทุกตัวในหมวดเห็นชื่อใหม่ทันที (ชี้ด้วย id)
func (r *PostgresRepository) Update(ctx context.Context, id string, req CategoryRequest) (Category, error) {
	_, err := r.db.Exec(ctx, `UPDATE product_categories SET name = $2, updated_at = now() WHERE id = $1`, id, req.cleanName())
	if err != nil {
		return Category{}, nameTakenOr(err)
	}
	var c Category
	err = r.db.QueryRow(ctx, selectCategories+` WHERE c.id = $1 GROUP BY c.id`, id).Scan(&c.ID, &c.Name, &c.ProductCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return Category{}, ErrNotFound
	}
	return c, err
}

// Delete ลบหมวด — สินค้าในหมวดนี้กลายเป็นไม่มีหมวด (ON DELETE SET NULL) ไม่ได้ลบสินค้า
func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.Exec(ctx, `DELETE FROM product_categories WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// nameTakenOr เปลี่ยน error "ชื่อซ้ำ" ของ Postgres เป็น ErrNameTaken นอกนั้นส่งต่อเหมือนเดิม
func nameTakenOr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return ErrNameTaken
	}
	return err
}
