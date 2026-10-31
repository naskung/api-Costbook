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
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) FindByEmail(ctx context.Context, email string) (User, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, email, first_name, last_name, password_hash
		FROM users
		WHERE email = $1`, email)
	return scanUser(row)
}

func (r *PostgresRepository) FindByID(ctx context.Context, id string) (User, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, email, first_name, last_name, password_hash
		FROM users
		WHERE id = $1`, id)
	return scanUser(row)
}

// scanUser อ่านผลจาก SELECT ใส่ User (ไม่เจอแถว → ErrNotFound)
func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}
