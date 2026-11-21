package quotation

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/naskung/api-Costbook/internal/utils"
)

// error ที่ handler แปลงเป็น status + error code
var (
	ErrNotFound         = errors.New("quotation not found")         // 404 quotation_not_found
	ErrProductNotFound  = errors.New("product not found")           // 404 product_not_found
	ErrNotForSale       = errors.New("product is not for sale")     // 409 product_not_for_sale (วัสดุ)
	ErrAlreadyCancelled = errors.New("quotation already cancelled") // 409 quotation_already_cancelled
)

// Repository คือรายการคำสั่งที่คุยกับ DB
// period = ช่วงวันที่ที่เลือก (นับรวมทั้ง 2 วัน)
type Repository interface {
	// limit / offset = โหลดทีละหน้า (limit 0 = ทั้งช่วง)
	List(ctx context.Context, period utils.DateRange, limit, offset int) ([]QuotationSummary, error)
	FindByID(ctx context.Context, id string) (Quotation, error)
	Create(ctx context.Context, req QuotationRequest) (Quotation, error)
	Cancel(ctx context.Context, id string) (Quotation, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// seller ข้อมูลร้าน ณ ตอนออกใบ (copy ไปเก็บในใบ) + รูปแบบร้าน
type seller struct {
	name, address, taxID, branch, phone string
	vatRegistered, pricesIncludeVAT     bool
}

// List ล่าสุดก่อน / id ต่อท้ายให้ลำดับตายตัว (โหลดทีละหน้าไม่ซ้ำ ไม่ข้าม) / LIMIT NULL = ไม่จำกัด
func (r *PostgresRepository) List(ctx context.Context, period utils.DateRange, limit, offset int) ([]QuotationSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, quotation_no, issue_date::text, valid_until::text, status, customer_name, total
		FROM quotations
		WHERE issue_date BETWEEN $1::date AND $2::date
		ORDER BY issue_date DESC, quotation_no DESC, id
		LIMIT NULLIF($3::int, 0) OFFSET $4`, period.From, period.To, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	quotations := []QuotationSummary{} // ไม่ใช้ nil → JSON จะได้ [] แทน null
	for rows.Next() {
		var s QuotationSummary
		err := rows.Scan(&s.ID, &s.QuotationNo, &s.IssueDate, &s.ValidUntil, &s.Status, &s.CustomerName, &s.Total)
		if err != nil {
			return nil, err
		}
		quotations = append(quotations, s)
	}
	return quotations, rows.Err()
}

func (r *PostgresRepository) FindByID(ctx context.Context, id string) (Quotation, error) {
	var q Quotation
	err := r.db.QueryRow(ctx, `
		SELECT id, quotation_no, issue_date::text, valid_until::text, status,
			seller_name, seller_address, seller_tax_id, seller_branch, seller_phone,
			customer_name, customer_address, customer_tax_id, customer_branch,
			note, with_vat, prices_include_vat, items_total, discount, subtotal, vat, total
		FROM quotations
		WHERE id = $1`, id).
		Scan(&q.ID, &q.QuotationNo, &q.IssueDate, &q.ValidUntil, &q.Status,
			&q.SellerName, &q.SellerAddress, &q.SellerTaxID, &q.SellerBranch, &q.SellerPhone,
			&q.CustomerName, &q.CustomerAddress, &q.CustomerTaxID, &q.CustomerBranch,
			&q.Note, &q.WithVAT, &q.PricesIncludeVAT, &q.ItemsTotal, &q.Discount, &q.Subtotal, &q.VAT, &q.Total)
	if errors.Is(err, pgx.ErrNoRows) {
		return Quotation{}, ErrNotFound
	}
	if err != nil {
		return Quotation{}, err
	}

	q.Items, err = r.items(ctx, id)
	return q, err
}

// Create ออกใบเสนอราคา ทำใน transaction เดียว ขั้นไหนพัง ยกเลิกทั้งหมด:
// 1. อ่านข้อมูลร้าน + รันเลขที่ใบ  2. บันทึกใบ + รายการสินค้า
// ไม่ตัดสต็อก ไม่บันทึกรายรับ (ยังไม่ได้ขายจริง)
func (r *PostgresRepository) Create(ctx context.Context, req QuotationRequest) (Quotation, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Quotation{}, err
	}
	defer tx.Rollback(ctx) // ถ้า Commit แล้ว คำสั่งนี้ไม่ทำอะไร

	shop, err := sellerInfo(ctx, tx)
	if err != nil {
		return Quotation{}, err
	}
	quotationNo, err := nextQuotationNo(ctx, tx, req.IssueDate)
	if err != nil {
		return Quotation{}, err
	}
	// ร้านจด VAT → คิด VAT จากยอดรวมทั้งใบหลังหักส่วนลด / ไม่จด → ไม่มี VAT
	amounts := utils.DocumentAmounts(itemsTotal(req.Items), req.Discount, shop.vatRegistered, shop.pricesIncludeVAT)
	quotationID, err := insertQuotation(ctx, tx, req, shop, quotationNo, amounts)
	if err != nil {
		return Quotation{}, err
	}
	for i, item := range req.Items {
		err = addItem(ctx, tx, quotationID, i+1, item)
		if err != nil {
			return Quotation{}, err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return Quotation{}, err
	}
	return r.FindByID(ctx, quotationID)
}

// Cancel ยกเลิกใบเสนอราคา (เปลี่ยนสถานะอย่างเดียว ไม่มีสต็อก / เงินให้คืน)
func (r *PostgresRepository) Cancel(ctx context.Context, id string) (Quotation, error) {
	result, err := r.db.Exec(ctx, `
		UPDATE quotations SET status = 'cancelled', cancelled_at = now(), updated_at = now()
		WHERE id = $1 AND status = 'issued'`, id)
	if err != nil {
		return Quotation{}, err
	}
	if result.RowsAffected() == 0 {
		return Quotation{}, r.notFoundOrCancelled(ctx, id)
	}
	return r.FindByID(ctx, id)
}

// sellerInfo อ่านข้อมูลร้าน + รูปแบบร้าน (แถว id = 1 มีเสมอ migration สร้างไว้ให้)
func sellerInfo(ctx context.Context, tx pgx.Tx) (seller, error) {
	var s seller
	err := tx.QueryRow(ctx, `
		SELECT name, address, tax_id, branch, phone, vat_registered, prices_include_vat
		FROM company
		WHERE id = 1`).
		Scan(&s.name, &s.address, &s.taxID, &s.branch, &s.phone, &s.vatRegistered, &s.pricesIncludeVAT)
	return s, err
}

// nextQuotationNo เพิ่มเลขของปีนั้นทีละ 1 (ปีแรกเริ่มที่ 1) ใน transaction → กดพร้อมกันเลขก็ไม่ซ้ำ
func nextQuotationNo(ctx context.Context, tx pgx.Tx, issueDate string) (string, error) {
	var year, number int
	err := tx.QueryRow(ctx, `
		INSERT INTO quotation_numbers (year, last_number)
		VALUES (EXTRACT(YEAR FROM $1::date)::int, 1)
		ON CONFLICT (year) DO UPDATE SET last_number = quotation_numbers.last_number + 1, updated_at = now()
		RETURNING year, last_number`, issueDate).Scan(&year, &number)
	return quotationNumber(year, number), err
}

// insertQuotation บันทึกหัวใบ คืน id ใบ
func insertQuotation(ctx context.Context, tx pgx.Tx, req QuotationRequest, shop seller, quotationNo string, m utils.DocAmounts) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO quotations (quotation_no, issue_date, valid_until,
			seller_name, seller_address, seller_tax_id, seller_branch, seller_phone,
			customer_name, customer_address, customer_tax_id, customer_branch,
			note, with_vat, prices_include_vat, items_total, discount, subtotal, vat, total)
		VALUES ($1, $2::date, $3::date, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		RETURNING id`,
		quotationNo, req.IssueDate, req.ValidUntil,
		shop.name, shop.address, shop.taxID, shop.branch, shop.phone,
		req.CustomerName, req.CustomerAddress, req.CustomerTaxID, req.CustomerBranch,
		req.Note, shop.vatRegistered, shop.pricesIncludeVAT && shop.vatRegistered,
		m.ItemsTotal, m.Discount, m.Subtotal, m.VAT, m.Total).
		Scan(&id)
	return id, err
}

// addItem บันทึกสินค้า 1 บรรทัด: เก็บชื่อ / หน่วย ณ วันที่ออก (วัสดุเสนอราคาไม่ได้ เพราะไม่ได้ขาย)
func addItem(ctx context.Context, tx pgx.Tx, quotationID string, lineNo int, item ItemRequest) error {
	var name, unit string
	var forSale bool
	err := tx.QueryRow(ctx, `SELECT name, unit, for_sale FROM products WHERE id = $1`, item.ProductID).
		Scan(&name, &unit, &forSale)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrProductNotFound
	}
	if err != nil {
		return err
	}
	if !forSale {
		return ErrNotForSale
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO quotation_items (quotation_id, line_no, product_id, name, unit, quantity, unit_price, amount)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		quotationID, lineNo, item.ProductID, name, unit, item.Quantity, item.UnitPrice,
		utils.LineAmount(item.Quantity, item.UnitPrice))
	return err
}

// notFoundOrCancelled ยกเลิกไม่ได้ เพราะไม่มีใบนี้ หรือยกเลิกไปแล้ว
func (r *PostgresRepository) notFoundOrCancelled(ctx context.Context, id string) error {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM quotations WHERE id = $1)`, id).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return ErrAlreadyCancelled
}

// items รายการสินค้าในใบ เรียงตามบรรทัด (สินค้าที่ถูกลบไปแล้ว product_id = "")
func (r *PostgresRepository) items(ctx context.Context, quotationID string) ([]QuotationItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, COALESCE(product_id::text, ''), name, unit, quantity, unit_price, amount
		FROM quotation_items
		WHERE quotation_id = $1
		ORDER BY line_no`, quotationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []QuotationItem{}
	for rows.Next() {
		var item QuotationItem
		err := rows.Scan(&item.ID, &item.ProductID, &item.Name, &item.Unit, &item.Quantity, &item.UnitPrice, &item.Amount)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
