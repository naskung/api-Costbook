package field

import (
	"context"
	"errors"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound ใช้บอก handler ว่าไม่เจอช่องนี้ (handler จะตอบ 404)
var ErrNotFound = errors.New("product field not found")

// ErrLayoutInvalid ลำดับฟอร์มที่ส่งมาไม่ครบ / เกิน / มีช่องที่ไม่มีจริง (handler จะตอบ 400 form_layout_invalid)
var ErrLayoutInvalid = errors.New("product form layout invalid")

// Repository คือรายการคำสั่งที่คุยกับ DB
// ทำเป็น interface เพื่อให้ตอน test ใช้ fakeRepository แทน DB จริงได้
type Repository interface {
	List(ctx context.Context) ([]ProductField, error)
	Create(ctx context.Context, req FieldRequest) (ProductField, error)
	Update(ctx context.Context, id string, req FieldRequest) (ProductField, error)
	Delete(ctx context.Context, id string) error
	Layout(ctx context.Context) ([]LayoutItem, error)
	SaveLayout(ctx context.Context, items []LayoutItem) error
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

// Create ช่องใหม่อยู่ท้ายฟอร์มเสมอ (sort_order = เลขมากสุดของทั้งฟอร์ม + 1 ไม่ใช้ค่าที่ส่งมา) แล้วค่อยลากย้ายทีหลัง
func (r *PostgresRepository) Create(ctx context.Context, req FieldRequest) (ProductField, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO product_fields (name, type, options, show_in_list, sort_order)
		VALUES ($1, $2, $3, $4, (
			SELECT COALESCE(MAX(sort_order), 0) + 1
			FROM (SELECT sort_order FROM product_fields UNION ALL SELECT sort_order FROM product_form_blocks) AS form_items
		))
		RETURNING `+fieldColumns,
		req.Name, req.Type, req.cleanOptions(), req.ShowInList)
	return scanField(row)
}

// Update แก้ชื่อ / ชนิด / ตัวเลือก / แสดงในตาราง — ไม่แตะตำแหน่ง (ย้ายตำแหน่งใช้ SaveLayout ทางเดียว)
func (r *PostgresRepository) Update(ctx context.Context, id string, req FieldRequest) (ProductField, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE product_fields
		SET name = $2, type = $3, options = $4, show_in_list = $5, updated_at = now()
		WHERE id = $1
		RETURNING `+fieldColumns,
		id, req.Name, req.Type, req.cleanOptions(), req.ShowInList)
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

// Layout ลำดับของทุกช่องในฟอร์มสินค้า (ช่องหลัก + ช่องที่ร้านเพิ่ม) เรียงจากบนลงล่าง — เลขเท่ากัน ช่องหลักขึ้นก่อน
func (r *PostgresRepository) Layout(ctx context.Context) ([]LayoutItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT key, sort_order FROM (
			SELECT key, sort_order, 0 AS kind FROM product_form_blocks
			UNION ALL
			SELECT id::text, sort_order, 1 FROM product_fields
		) AS form_items
		ORDER BY sort_order, kind, key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []LayoutItem{}
	for rows.Next() {
		var item LayoutItem
		if err := rows.Scan(&item.Key, &item.SortOrder); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// SaveLayout บันทึกลำดับใหม่ทั้งฟอร์มใน transaction เดียว
// ต้องส่งครบพอดี: ช่องหลักทุกช่อง (LayoutRequest.Validate เช็คแล้ว) + ช่องที่ร้านเพิ่มทุกช่อง ไม่งั้น ErrLayoutInvalid (ไม่แก้อะไรเลย)
func (r *PostgresRepository) SaveLayout(ctx context.Context, items []LayoutItem) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var fieldCount int
	err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM (SELECT id FROM product_fields FOR UPDATE) AS locked`).Scan(&fieldCount)
	if err != nil {
		return err
	}
	if len(items) != len(FormBlocks)+fieldCount {
		return ErrLayoutInvalid
	}
	for _, item := range items {
		sql := `UPDATE product_fields SET sort_order = $2, updated_at = now() WHERE id::text = $1`
		if slices.Contains(FormBlocks, item.Key) {
			sql = `UPDATE product_form_blocks SET sort_order = $2, updated_at = now() WHERE key = $1`
		}
		result, err := tx.Exec(ctx, sql, item.Key, item.SortOrder)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return ErrLayoutInvalid
		}
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
