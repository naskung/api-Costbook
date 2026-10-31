package product

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound ใช้บอก handler ว่าไม่เจอสินค้า (handler จะตอบ 404)
var ErrNotFound = errors.New("product not found")

// ErrInUse สินค้าเคยอยู่ในใบกำกับแล้ว ลบไม่ได้ (handler จะตอบ 409)
var ErrInUse = errors.New("product is used in an invoice")

// ErrImageNotFound สินค้ายังไม่มีรูป (handler จะตอบ 404)
var ErrImageNotFound = errors.New("product image not found")

// ErrCodeTaken รหัส / บาร์โค้ดซ้ำกับสินค้าตัวอื่น (handler จะตอบ 409)
var ErrCodeTaken = errors.New("product code is already used")

// uniqueViolation code ของ Postgres เมื่อบันทึกค่าที่ห้ามซ้ำแล้วซ้ำ
const uniqueViolation = "23505"

// Repository คือรายการคำสั่งที่คุยกับ DB
// ทำเป็น interface เพื่อให้ตอน test ใช้ fakeRepository แทน DB จริงได้
type Repository interface {
	List(ctx context.Context) ([]Product, error)
	FindByID(ctx context.Context, id string) (Product, error)
	Create(ctx context.Context, req ProductRequest) (Product, error)
	Update(ctx context.Context, id string, req ProductRequest) (Product, error)
	Delete(ctx context.Context, id string) error
	FindImage(ctx context.Context, id string) (Image, error)
	SaveImage(ctx context.Context, id string, image Image) (Product, error)
	DeleteImage(ctx context.Context, id string) (Product, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// productColumns column ที่ใช้อ่านสินค้า เรียงตรงกับ scanProduct
const productColumns = `id, code, name, category, description, unit, cost_price, sell_price,
	stock, min_stock, batch_quantity, track_stock, custom_fields, image_version`

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
		INSERT INTO products (code, name, category, description, unit, cost_price, sell_price,
			min_stock, track_stock, custom_fields)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING `+productColumns,
		req.Code, req.Name, req.Category, req.Description, req.Unit, req.CostPrice, req.SellPrice,
		req.MinStock, req.trackStock(), req.customFields())
	return codeTakenOr(scanProduct(row))
}

// Update แก้สินค้า track_stock / custom_fields ที่ไม่ได้ส่งมา (nil → NULL) ใช้ค่าเดิม (COALESCE)
func (r *PostgresRepository) Update(ctx context.Context, id string, req ProductRequest) (Product, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE products
		SET code = $2, name = $3, category = $4, description = $5, unit = $6, cost_price = $7, sell_price = $8,
			min_stock = $9, track_stock = COALESCE($10, track_stock), custom_fields = COALESCE($11, custom_fields),
			updated_at = now()
		WHERE id = $1
		RETURNING `+productColumns,
		id, req.Code, req.Name, req.Category, req.Description, req.Unit, req.CostPrice, req.SellPrice,
		req.MinStock, req.TrackStock, req.CustomFields)
	return codeTakenOr(scanProduct(row))
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

// FindImage อ่านรูปของสินค้า (ไม่มีสินค้า = ErrNotFound, มีสินค้าแต่ไม่มีรูป = ErrImageNotFound)
func (r *PostgresRepository) FindImage(ctx context.Context, id string) (Image, error) {
	var image Image
	err := r.db.QueryRow(ctx, `SELECT image, image_content_type FROM products WHERE id = $1`, id).
		Scan(&image.Data, &image.ContentType)
	if errors.Is(err, pgx.ErrNoRows) {
		return Image{}, ErrNotFound
	}
	if err != nil {
		return Image{}, err
	}
	if len(image.Data) == 0 {
		return Image{}, ErrImageNotFound
	}
	return image, nil
}

// SaveImage เก็บรูปใหม่ทับรูปเดิม + ตั้ง image_version เป็นเวลาตอนนี้ (มิลลิวินาที)
func (r *PostgresRepository) SaveImage(ctx context.Context, id string, image Image) (Product, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE products
		SET image = $2, image_content_type = $3,
			image_version = (EXTRACT(EPOCH FROM clock_timestamp()) * 1000)::BIGINT, updated_at = now()
		WHERE id = $1
		RETURNING `+productColumns, id, image.Data, image.ContentType)
	return scanProduct(row)
}

// DeleteImage ลบรูปของสินค้า (image_version กลับเป็น 0)
func (r *PostgresRepository) DeleteImage(ctx context.Context, id string) (Product, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE products
		SET image = NULL, image_content_type = '', image_version = 0, updated_at = now()
		WHERE id = $1
		RETURNING `+productColumns, id)
	return scanProduct(row)
}

// scanProduct อ่านผลจาก SELECT / RETURNING ใส่ Product
func scanProduct(row pgx.Row) (Product, error) {
	var p Product
	err := row.Scan(&p.ID, &p.Code, &p.Name, &p.Category, &p.Description, &p.Unit, &p.CostPrice, &p.SellPrice,
		&p.Stock, &p.MinStock, &p.BatchQuantity, &p.TrackStock, &p.CustomFields, &p.ImageVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	return p, err
}

// codeTakenOr เปลี่ยน error "รหัสซ้ำ" ของ Postgres เป็น ErrCodeTaken นอกนั้นส่งต่อเหมือนเดิม
func codeTakenOr(p Product, err error) (Product, error) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return Product{}, ErrCodeTaken
	}
	return p, err
}
