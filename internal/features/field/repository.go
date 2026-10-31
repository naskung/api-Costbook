package field

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound ใช้บอก handler ว่าไม่เจอช่องนี้ (handler จะตอบ 404)
var ErrNotFound = errors.New("product field not found")

// Repository คือรายการคำสั่งที่คุยกับ DB
// ทำเป็น interface เพื่อให้ตอน test ใช้ fakeRepository แทน DB จริงได้
type Repository interface {
	List(ctx context.Context) ([]ProductField, error)
	Create(ctx context.Context, req FieldRequest) (ProductField, error)
	Update(ctx context.Context, id string, req FieldRequest) (ProductField, error)
	Delete(ctx context.Context, id string) error
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// fieldColumns column ที่ใช้อ่านช่อง เรียงตรงกับ scanField
const fieldColumns = "id, name, type, options, show_in_list, sort_order"

func (r *PostgresRepository) List(ctx context.Context) ([]ProductField, error) {
	rows, err := r.db.Query(ctx, `SELECT `+fieldColumns+` FROM product_fields ORDER BY sort_order, created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fields := []ProductField{} // ไม่ใช้ nil → JSON จะได้ [] แทน null
	for rows.Next() {
		f, err := scanField(rows)
		if err != nil {
			return nil, err
		}
		fields = append(fields, f)
	}
	return fields, rows.Err()
}

func (r *PostgresRepository) Create(ctx context.Context, req FieldRequest) (ProductField, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO product_fields (name, type, options, show_in_list, sort_order)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+fieldColumns,
		req.Name, req.Type, req.cleanOptions(), req.ShowInList, req.SortOrder)
	return scanField(row)
}

func (r *PostgresRepository) Update(ctx context.Context, id string, req FieldRequest) (ProductField, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE product_fields
		SET name = $2, type = $3, options = $4, show_in_list = $5, sort_order = $6, updated_at = now()
		WHERE id = $1
		RETURNING `+fieldColumns,
		id, req.Name, req.Type, req.cleanOptions(), req.ShowInList, req.SortOrder)
	return scanField(row)
}

// Delete ลบช่อง + ลบค่าของช่องนี้ออกจากสินค้าทุกตัว ทำใน transaction เดียว
func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // ถ้า Commit แล้ว คำสั่งนี้ไม่ทำอะไร

	result, err := tx.Exec(ctx, `DELETE FROM product_fields WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	_, err = tx.Exec(ctx, `UPDATE products SET custom_fields = custom_fields - $1::text WHERE custom_fields ? $1::text`, id)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// scanField อ่านผลจาก SELECT / RETURNING ใส่ ProductField
func scanField(row pgx.Row) (ProductField, error) {
	var f ProductField
	err := row.Scan(&f.ID, &f.Name, &f.Type, &f.Options, &f.ShowInList, &f.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProductField{}, ErrNotFound
	}
	if f.Options == nil {
		f.Options = []string{}
	}
	return f, err
}
