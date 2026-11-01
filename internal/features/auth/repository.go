package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound ใช้บอก handler ว่าไม่เจอผู้ใช้
var ErrNotFound = errors.New("user not found")

// Repository คือรายการคำสั่งที่คุยกับ DB
// ทำเป็น interface เพื่อให้ตอน test ใช้ fakeRepository แทน DB จริงได้
type Repository interface {
	FindByEmail(ctx context.Context, email string) (User, error)
	FindByID(ctx context.Context, id string) (User, error)
	Status(ctx context.Context, id string) (role string, active bool, err error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const userColumns = `id, email, first_name, last_name, role, avatar_version, active, password_hash`

func (r *PostgresRepository) FindByEmail(ctx context.Context, email string) (User, error) {
	row := r.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email)
	return scanUser(row)
}

func (r *PostgresRepository) FindByID(ctx context.Context, id string) (User, error) {
	row := r.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id)
	return scanUser(row)
}

// Status role + ยังใช้งานได้ไหม (middleware.Authenticated เรียกทุก request) — ไม่เจอผู้ใช้ = ไม่ active ไม่ใช่ error
func (r *PostgresRepository) Status(ctx context.Context, id string) (string, bool, error) {
	var role string
	var active bool
	err := r.db.QueryRow(ctx, `SELECT role, active FROM users WHERE id = $1`, id).Scan(&role, &active)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	return role, active, err
}

// scanUser อ่านผลจาก SELECT ใส่ User (ไม่เจอแถว → ErrNotFound)
func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.Role, &u.AvatarVersion, &u.Active, &u.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}
