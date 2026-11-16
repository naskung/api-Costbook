package transaction

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/naskung/api-Costbook/internal/utils"
)

// ErrNotFound ใช้บอก handler ว่าไม่เจอรายการ (handler จะตอบ 404)
var ErrNotFound = errors.New("transaction not found")

// ErrFromInvoice รายการนี้มาจากใบกำกับ แก้ / ลบตรง ๆ ไม่ได้ (handler จะตอบ 409)
var ErrFromInvoice = errors.New("transaction comes from an invoice")

// Repository คือรายการคำสั่งที่คุยกับ DB
// period = ช่วงวันที่ที่เลือก เช่น 2026-11-01 ถึง 2026-11-16 (นับรวมทั้ง 2 วัน)
type Repository interface {
	// limit / offset = โหลดทีละหน้า (limit 0 = ทั้งช่วง)
	List(ctx context.Context, period utils.DateRange, limit, offset int) ([]Transaction, error)
	Amounts(ctx context.Context, period utils.DateRange) (Amounts, error)
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

// List ล่าสุดก่อน / id ต่อท้ายให้ลำดับตายตัว (โหลดทีละหน้าไม่ซ้ำ ไม่ข้าม) / LIMIT NULL = ไม่จำกัด
func (r *PostgresRepository) List(ctx context.Context, period utils.DateRange, limit, offset int) ([]Transaction, error) {
	rows, err := r.db.Query(ctx, transactionSelect+`
		WHERE t.date BETWEEN $1::date AND $2::date
		ORDER BY t.date DESC, t.created_at DESC, t.id
		LIMIT NULLIF($3::int, 0) OFFSET $4`, period.From, period.To, limit, offset)
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

// AmountsSQL ยอดของช่วงวันที่ที่ใช้คิดกำไร ($1 = วันแรก, $2 = วันสุดท้าย นับรวม) เรียงตรงกับ Amounts
// income / expense / รายรับที่บันทึกเอง (ไม่มี invoice_id) / ซื้อวัตถุดิบ / ยอดขายก่อน VAT / ต้นทุนสินค้าที่ขาย
// หน้าภาพรวมใช้ SQL ตัวนี้ด้วย
const AmountsSQL = `
	SELECT
		COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'income'), 0),
		COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'expense'), 0),
		COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'income' AND t.invoice_id IS NULL), 0),
		COALESCE(SUM(t.amount) FILTER (WHERE t.type = 'expense' AND t.category = '` + CategoryMaterial + `'), 0),
		COALESCE((SELECT SUM(i.subtotal) FROM invoices i
			WHERE i.status = 'issued' AND i.issue_date BETWEEN $1::date AND $2::date), 0),
		COALESCE((SELECT SUM(ii.quantity * ii.cost_price) FROM invoice_items ii JOIN invoices i ON i.id = ii.invoice_id
			WHERE i.status = 'issued' AND i.issue_date BETWEEN $1::date AND $2::date), 0)
	FROM transactions t
	WHERE t.date BETWEEN $1::date AND $2::date`

// ScanAmounts อ่านผลของ AmountsSQL
func ScanAmounts(row pgx.Row) (Amounts, error) {
	var a Amounts
	err := row.Scan(&a.Income, &a.Expense, &a.OtherIncome, &a.MaterialExpense, &a.Sales, &a.CostOfSales)
	return a, err
}

func (r *PostgresRepository) Amounts(ctx context.Context, period utils.DateRange) (Amounts, error) {
	return ScanAmounts(r.db.QueryRow(ctx, AmountsSQL, period.From, period.To))
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
