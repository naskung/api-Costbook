package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/naskung/api-Costbook/internal/features/auth"
)

var (
	// ErrNotFound ไม่มีผู้ใช้นี้
	ErrNotFound = errors.New("user not found")
	// ErrEmailTaken อีเมลนี้มีผู้ใช้แล้ว
	ErrEmailTaken = errors.New("email already used")
	// ErrLastOwner แก้แล้วร้านจะไม่เหลือเจ้าของที่ใช้งานได้ (เปลี่ยน role / ปิดบัญชีเจ้าของคนสุดท้าย)
	ErrLastOwner = errors.New("shop needs at least one active owner")
	// ErrAvatarNotFound ผู้ใช้ยังไม่มีรูปโปรไฟล์
	ErrAvatarNotFound = errors.New("avatar not found")
)

// Repository คำสั่งที่คุยกับตาราง users (รหัสผ่านรับ / คืนเป็น hash ของ bcrypt เสมอ)
type Repository interface {
	List(ctx context.Context) ([]User, error)
	Find(ctx context.Context, id string) (User, error)
	PasswordHash(ctx context.Context, id string) (string, error)
	Create(ctx context.Context, req CreateRequest, passwordHash string) (User, error)
	Update(ctx context.Context, id string, req UpdateRequest) (User, error)
	UpdateProfile(ctx context.Context, id string, req ProfileRequest) (User, error)
	UpdatePassword(ctx context.Context, id, passwordHash string) error
	FindAvatar(ctx context.Context, id string) (Avatar, error)
	SaveAvatar(ctx context.Context, id string, avatar Avatar) (User, error)
	DeleteAvatar(ctx context.Context, id string) (User, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const userColumns = `id, email, first_name, last_name, role, active, avatar_version, created_at`

// List เจ้าของก่อน แล้วผู้จัดการ พนักงาน / role เดียวกันเรียงตามชื่อ
func (r *PostgresRepository) List(ctx context.Context) ([]User, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+userColumns+`
		FROM users
		ORDER BY array_position(ARRAY['owner', 'manager', 'staff'], role), active DESC, first_name COLLATE "C", email`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (User, error) { return scanUser(row) })
}

func (r *PostgresRepository) Find(ctx context.Context, id string) (User, error) {
	return scanUser(r.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id))
}

func (r *PostgresRepository) PasswordHash(ctx context.Context, id string) (string, error) {
	var hash string
	err := r.db.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, id).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return hash, err
}

// Create อีเมลซ้ำ (unique ใน DB) → ErrEmailTaken
func (r *PostgresRepository) Create(ctx context.Context, req CreateRequest, passwordHash string) (User, error) {
	user, err := scanUser(r.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, first_name, last_name, role)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+userColumns,
		normalizeEmail(req.Email), passwordHash, strings.TrimSpace(req.FirstName), strings.TrimSpace(req.LastName), req.Role))
	if isUniqueViolation(err) {
		return User{}, ErrEmailTaken
	}
	return user, err
}

// Update แก้ชื่อ / role / เปิด-ปิดบัญชี — ถ้าทำให้ร้านไม่เหลือเจ้าของที่ใช้งานได้ → ErrLastOwner (ไม่แก้อะไรเลย)
// ล็อกแถวเจ้าของทุกคนก่อนนับ กันเจ้าของ 2 คนปลดกันพร้อมกันจนไม่เหลือใคร
func (r *PostgresRepository) Update(ctx context.Context, id string, req UpdateRequest) (User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)

	current, err := scanUser(tx.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1 FOR UPDATE`, id))
	if err != nil {
		return User{}, err
	}
	active := current.Active
	if req.Active != nil {
		active = *req.Active
	}
	if stopsBeingOwner(current, req.Role, active) {
		var otherOwners int
		err = tx.QueryRow(ctx, `
			SELECT count(*) FROM (
				SELECT id FROM users WHERE role = 'owner' AND active AND id <> $1 FOR UPDATE
			) owners`, id).Scan(&otherOwners)
		if err != nil {
			return User{}, err
		}
		if otherOwners == 0 {
			return User{}, ErrLastOwner
		}
	}

	user, err := scanUser(tx.QueryRow(ctx, `
		UPDATE users SET first_name = $2, last_name = $3, role = $4, active = $5, updated_at = now()
		WHERE id = $1
		RETURNING `+userColumns,
		id, strings.TrimSpace(req.FirstName), strings.TrimSpace(req.LastName), req.Role, active))
	if err != nil {
		return User{}, err
	}
	return user, tx.Commit(ctx)
}

// stopsBeingOwner เจ้าของที่ใช้งานอยู่ กำลังจะไม่ใช่เจ้าของที่ใช้งานได้ (เปลี่ยน role หรือถูกปิดบัญชี)
func stopsBeingOwner(current User, newRole string, newActive bool) bool {
	isOwnerNow := current.Role == auth.RoleOwner && current.Active
	return isOwnerNow && (newRole != auth.RoleOwner || !newActive)
}

func (r *PostgresRepository) UpdateProfile(ctx context.Context, id string, req ProfileRequest) (User, error) {
	return scanUser(r.db.QueryRow(ctx, `
		UPDATE users SET first_name = $2, last_name = $3, updated_at = now()
		WHERE id = $1
		RETURNING `+userColumns,
		id, strings.TrimSpace(req.FirstName), strings.TrimSpace(req.LastName)))
}

func (r *PostgresRepository) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	tag, err := r.db.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`, id, passwordHash)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// FindAvatar ไม่มีผู้ใช้ = ErrNotFound, มีแต่ไม่มีรูป = ErrAvatarNotFound
func (r *PostgresRepository) FindAvatar(ctx context.Context, id string) (Avatar, error) {
	var avatar Avatar
	err := r.db.QueryRow(ctx, `SELECT avatar, avatar_content_type FROM users WHERE id = $1`, id).Scan(&avatar.Data, &avatar.ContentType)
	if errors.Is(err, pgx.ErrNoRows) {
		return Avatar{}, ErrNotFound
	}
	if err != nil {
		return Avatar{}, err
	}
	if len(avatar.Data) == 0 {
		return Avatar{}, ErrAvatarNotFound
	}
	return avatar, nil
}

// SaveAvatar avatar_version = เวลาที่เปลี่ยนรูป (มิลลิวินาที) → URL รูปเปลี่ยน browser โหลดรูปใหม่
func (r *PostgresRepository) SaveAvatar(ctx context.Context, id string, avatar Avatar) (User, error) {
	return scanUser(r.db.QueryRow(ctx, `
		UPDATE users SET avatar = $2, avatar_content_type = $3, avatar_version = $4, updated_at = now()
		WHERE id = $1
		RETURNING `+userColumns,
		id, avatar.Data, avatar.ContentType, time.Now().UnixMilli()))
}

func (r *PostgresRepository) DeleteAvatar(ctx context.Context, id string) (User, error) {
	return scanUser(r.db.QueryRow(ctx, `
		UPDATE users SET avatar = NULL, avatar_content_type = '', avatar_version = 0, updated_at = now()
		WHERE id = $1
		RETURNING `+userColumns, id))
}

// scanUser อ่าน 1 แถว (ลำดับตาม userColumns) — ไม่เจอแถว → ErrNotFound
func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.Role, &u.Active, &u.AvatarVersion, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

// isUniqueViolation error จาก PostgreSQL ว่าค่าซ้ำกับที่มีอยู่ (code 23505)
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
