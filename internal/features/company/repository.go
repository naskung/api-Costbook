package company

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrLogoNotFound ใช้บอก handler ว่ายังไม่มีโลโก้ (handler จะตอบ 404)
var ErrLogoNotFound = errors.New("logo not found")

// ErrCompanyMissing ไม่เจอแถวข้อมูลร้าน (ปกติ migration สร้างไว้ให้แล้ว)
var ErrCompanyMissing = errors.New("company row not found: run migrations/000002_create_company.up.sql")

// Repository คือรายการคำสั่งที่คุยกับ DB
// ทำเป็น interface เพื่อให้ตอน test ใช้ fakeRepository แทน DB จริงได้
type Repository interface {
	Get(ctx context.Context) (Company, error)
	Update(ctx context.Context, req CompanyRequest) (Company, error)
	FindLogo(ctx context.Context) (Logo, error)
	SaveLogo(ctx context.Context, logo Logo) (Company, error)
	DeleteLogo(ctx context.Context) error
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Get(ctx context.Context) (Company, error) {
	row := r.db.QueryRow(ctx, `
		SELECT name, address, tax_id, branch, phone, logo IS NOT NULL
		FROM company
		WHERE id = 1`)
	return scanCompany(row)
}

func (r *PostgresRepository) Update(ctx context.Context, req CompanyRequest) (Company, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE company
		SET name = $1, address = $2, tax_id = $3, branch = $4, phone = $5, updated_at = now()
		WHERE id = 1
		RETURNING name, address, tax_id, branch, phone, logo IS NOT NULL`,
		req.Name, req.Address, req.TaxID, req.Branch, req.Phone)
	return scanCompany(row)
}

func (r *PostgresRepository) FindLogo(ctx context.Context) (Logo, error) {
	var logo Logo
	err := r.db.QueryRow(ctx, `
		SELECT logo, logo_content_type
		FROM company
		WHERE id = 1 AND logo IS NOT NULL`).
		Scan(&logo.Data, &logo.ContentType)
	if errors.Is(err, pgx.ErrNoRows) {
		return Logo{}, ErrLogoNotFound
	}
	return logo, err
}

func (r *PostgresRepository) SaveLogo(ctx context.Context, logo Logo) (Company, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE company
		SET logo = $1, logo_content_type = $2, updated_at = now()
		WHERE id = 1
		RETURNING name, address, tax_id, branch, phone, logo IS NOT NULL`,
		logo.Data, logo.ContentType)
	return scanCompany(row)
}

func (r *PostgresRepository) DeleteLogo(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `
		UPDATE company
		SET logo = NULL, logo_content_type = NULL, updated_at = now()
		WHERE id = 1`)
	return err
}

// scanCompany อ่านผลจาก SELECT / RETURNING ใส่ Company
func scanCompany(row pgx.Row) (Company, error) {
	var c Company
	err := row.Scan(&c.Name, &c.Address, &c.TaxID, &c.Branch, &c.Phone, &c.HasLogo)
	if errors.Is(err, pgx.ErrNoRows) {
		return Company{}, ErrCompanyMissing
	}
	return c, err
}
