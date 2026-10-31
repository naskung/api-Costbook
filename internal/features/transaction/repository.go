package transaction

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound ใช้บอก handler ว่าไม่เจอรายการ (handler จะตอบ 404)
var ErrNotFound = errors.New("transaction not found")

// ErrFromInvoice รายการนี้มาจากใบกำกับ แก้ / ลบตรง ๆ ไม่ได้ (handler จะตอบ 409)
var ErrFromInvoice = errors.New("transaction comes from an invoice")

// Repository คือรายการคำสั่งที่คุยกับ DB
// monthStart = วันแรกของเดือน เช่น "2026-10-01"
type Repository interface {
	List(ctx context.Context, monthStart string) ([]Transaction, error)
	MonthTotals(ctx context.Context, monthStart string) (income, expense float64, err error)
	Create(ctx context.Context, req TransactionRequest) (Transaction, error)
	Update(ctx context.Context, id string, req TransactionRequest) (Transaction, error)
	Delete(ctx context.Context, id string) error
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// transactionSelect อ่านรายการพร้อมเลขที่ใบกำกับ เรียงตรงกับ scanTransaction
const transactionSelect = `
	SELECT t.id, t.type, t.category, t.amount, t.date::text, t.note,
		COALESCE(t.invoice_id::text, ''), COALESCE(i.invoice_no, '')
	FROM transactions t
	LEFT JOIN invoices i ON i.id = t.invoice_id`

func (r *PostgresRepository) List(ctx context.Context, monthStart string) ([]Transaction, error) {
	rows, err := r.db.Query(ctx, transactionSelect+`
		WHERE t.date >= $1::date AND t.date < $1::date + interval '1 month'
		ORDER BY t.date DESC, t.created_at DESC`, monthStart)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transactions := []Transaction{} // ไม่ใช้ nil → JSON จะได้ [] แทน null
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}
	return transactions, rows.Err()
}

func (r *PostgresRepository) MonthTotals(ctx context.Context, monthStart string) (float64, float64, error) {
	var income, expense float64
	err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE type = 'income'), 0),
			COALESCE(SUM(amount) FILTER (WHERE type = 'expense'), 0)
		FROM transactions
		WHERE date >= $1::date AND date < $1::date + interval '1 month'`, monthStart).
		Scan(&income, &expense)
	return income, expense, err
}

func (r *PostgresRepository) Create(ctx context.Context, req TransactionRequest) (Transaction, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO transactions (type, category, amount, date, note)
		VALUES ($1, $2, $3, $4::date, $5)
		RETURNING id`,
		req.Type, req.Category, req.Amount, req.Date, req.Note).Scan(&id)
	if err != nil {
		return Transaction{}, err
	}
	return r.findByID(ctx, id)
}

// Update แก้ได้เฉพาะรายการที่ไม่ได้มาจากใบกำกับ (invoice_id IS NULL)
func (r *PostgresRepository) Update(ctx context.Context, id string, req TransactionRequest) (Transaction, error) {
	result, err := r.db.Exec(ctx, `
		UPDATE transactions
		SET type = $2, category = $3, amount = $4, date = $5::date, note = $6, updated_at = now()
		WHERE id = $1 AND invoice_id IS NULL`,
		id, req.Type, req.Category, req.Amount, req.Date, req.Note)
	if err != nil {
		return Transaction{}, err
	}
	if result.RowsAffected() == 0 {
		return Transaction{}, r.notFoundOrFromInvoice(ctx, id)
	}
	return r.findByID(ctx, id)
}

// Delete ลบได้เฉพาะรายการที่ไม่ได้มาจากใบกำกับ (invoice_id IS NULL)
func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.Exec(ctx, `DELETE FROM transactions WHERE id = $1 AND invoice_id IS NULL`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return r.notFoundOrFromInvoice(ctx, id)
	}
	return nil
}

func (r *PostgresRepository) findByID(ctx context.Context, id string) (Transaction, error) {
	t, err := scanTransaction(r.db.QueryRow(ctx, transactionSelect+` WHERE t.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Transaction{}, ErrNotFound
	}
	return t, err
}

// notFoundOrFromInvoice แก้ / ลบไม่ได้ เพราะไม่มีรายการนี้ หรือรายการมาจากใบกำกับ
func (r *PostgresRepository) notFoundOrFromInvoice(ctx context.Context, id string) error {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM transactions WHERE id = $1)`, id).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return ErrFromInvoice
}

// scanTransaction อ่านผลจาก transactionSelect ใส่ Transaction
func scanTransaction(row pgx.Row) (Transaction, error) {
	var t Transaction
	err := row.Scan(&t.ID, &t.Type, &t.Category, &t.Amount, &t.Date, &t.Note, &t.InvoiceID, &t.InvoiceNo)
	return t, err
}
