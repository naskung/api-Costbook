package invoice

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/naskung/api-Costbook/internal/features/stock"
)

// error ที่ handler แปลงเป็น status + error code
var (
	ErrNotFound           = errors.New("invoice not found")                     // 404 invoice_not_found
	ErrProductNotFound    = errors.New("product not found")                     // 404 product_not_found
	ErrStockNotEnough     = errors.New("stock not enough")                      // 409 stock_not_enough
	ErrNotForSale         = errors.New("product is not for sale")               // 409 product_not_for_sale (วัสดุ)
	ErrAlreadyCancelled   = errors.New("invoice already cancelled")             // 409 invoice_already_cancelled
	ErrSellerTaxIDMissing = errors.New("company tax id is required to invoice") // 409 company_tax_id_required
	ErrCustomerRequired   = errors.New("tax invoice needs a customer name")     // 400 invoice_customer_required
)

// Repository คือรายการคำสั่งที่คุยกับ DB
// monthStart = วันแรกของเดือน เช่น "2026-10-01"
type Repository interface {
	// limit / offset = โหลดทีละหน้า (limit 0 = ทั้งเดือน)
	List(ctx context.Context, monthStart string, limit, offset int) ([]InvoiceSummary, error)
	FindByID(ctx context.Context, id string) (Invoice, error)
	Create(ctx context.Context, req InvoiceRequest) (Invoice, error)
	Cancel(ctx context.Context, id string) (Invoice, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// seller ข้อมูลร้าน ณ ตอนออกใบ (copy ไปเก็บในเอกสาร) + รูปแบบร้าน
type seller struct {
	name, address, taxID, branch, phone string
	vatRegistered, pricesIncludeVAT     bool
}

// List ล่าสุดก่อน / id ต่อท้ายให้ลำดับตายตัว (โหลดทีละหน้าไม่ซ้ำ ไม่ข้าม) / LIMIT NULL = ไม่จำกัด
func (r *PostgresRepository) List(ctx context.Context, monthStart string, limit, offset int) ([]InvoiceSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, doc_type, invoice_no, issue_date::text, status, customer_name, total
		FROM invoices
		WHERE issue_date >= $1::date AND issue_date < $1::date + interval '1 month'
		ORDER BY issue_date DESC, invoice_no DESC, id
		LIMIT NULLIF($2::int, 0) OFFSET $3`, monthStart, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	invoices := []InvoiceSummary{} // ไม่ใช้ nil → JSON จะได้ [] แทน null
	for rows.Next() {
		var s InvoiceSummary
		err := rows.Scan(&s.ID, &s.DocType, &s.InvoiceNo, &s.IssueDate, &s.Status, &s.CustomerName, &s.Total)
		if err != nil {
			return nil, err
		}
		invoices = append(invoices, s)
	}
	return invoices, rows.Err()
}

func (r *PostgresRepository) FindByID(ctx context.Context, id string) (Invoice, error) {
	var inv Invoice
	err := r.db.QueryRow(ctx, `
		SELECT id, doc_type, invoice_no, issue_date::text, status,
			seller_name, seller_address, seller_tax_id, seller_branch, seller_phone,
			customer_name, customer_address, customer_tax_id, customer_branch,
			note, prices_include_vat, items_total, discount, subtotal, vat, total
		FROM invoices
		WHERE id = $1`, id).
		Scan(&inv.ID, &inv.DocType, &inv.InvoiceNo, &inv.IssueDate, &inv.Status,
			&inv.SellerName, &inv.SellerAddress, &inv.SellerTaxID, &inv.SellerBranch, &inv.SellerPhone,
			&inv.CustomerName, &inv.CustomerAddress, &inv.CustomerTaxID, &inv.CustomerBranch,
			&inv.Note, &inv.PricesIncludeVAT, &inv.ItemsTotal, &inv.Discount, &inv.Subtotal, &inv.VAT, &inv.Total)
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, ErrNotFound
	}
	if err != nil {
		return Invoice{}, err
	}

	inv.Items, err = r.items(ctx, id)
	return inv, err
}

// Create ออกเอกสารขาย ทำใน transaction เดียว ขั้นไหนพัง ยกเลิกทั้งหมด:
// 1. เลือกชนิดเอกสารจากรูปแบบร้าน + รันเลขที่ใบ  2. บันทึกใบ + รายการสินค้า
// 3. ตัด stock + บันทึกประวัติ (เฉพาะสินค้าที่นับสต็อก)  4. บันทึกรายรับ
func (r *PostgresRepository) Create(ctx context.Context, req InvoiceRequest) (Invoice, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Invoice{}, err
	}
	defer tx.Rollback(ctx) // ถ้า Commit แล้ว คำสั่งนี้ไม่ทำอะไร

	shop, err := sellerInfo(ctx, tx)
	if err != nil {
		return Invoice{}, err
	}
	docType := docTypeFor(shop.vatRegistered)
	if docType == docTaxInvoice && shop.taxID == "" {
		return Invoice{}, ErrSellerTaxIDMissing
	}
	if docType == docTaxInvoice && strings.TrimSpace(req.CustomerName) == "" {
		return Invoice{}, ErrCustomerRequired
	}

	invoiceNo, err := nextInvoiceNo(ctx, tx, docType, req.IssueDate)
	if err != nil {
		return Invoice{}, err
	}
	amounts := documentAmounts(req.Items, req.Discount, docType, shop.pricesIncludeVAT)
	invoiceID, err := insertInvoice(ctx, tx, req, shop, docType, invoiceNo, amounts)
	if err != nil {
		return Invoice{}, err
	}
	for i, item := range req.Items {
		err = addItem(ctx, tx, invoiceID, i+1, item)
		if err != nil {
			return Invoice{}, err
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO transactions (type, category, amount, date, note, invoice_id)
		VALUES ('income', 'sales', $1, $2::date, $3, $4)`, amounts.total, req.IssueDate, invoiceNo, invoiceID)
	if err != nil {
		return Invoice{}, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return Invoice{}, err
	}
	return r.FindByID(ctx, invoiceID)
}

// Cancel ยกเลิกใบกำกับ ทำใน transaction เดียว: เปลี่ยนสถานะ + คืน stock + ลบรายรับของใบนี้
func (r *PostgresRepository) Cancel(ctx context.Context, id string) (Invoice, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Invoice{}, err
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx, `
		UPDATE invoices SET status = 'cancelled', cancelled_at = now(), updated_at = now()
		WHERE id = $1 AND status = 'issued'`, id)
	if err != nil {
		return Invoice{}, err
	}
	if result.RowsAffected() == 0 {
		return Invoice{}, r.notFoundOrCancelled(ctx, id)
	}

	err = returnStock(ctx, tx, id)
	if err != nil {
		return Invoice{}, err
	}
	err = stock.ReturnMaterials(ctx, tx, id)
	if err != nil {
		return Invoice{}, err
	}
	_, err = tx.Exec(ctx, `DELETE FROM transactions WHERE invoice_id = $1`, id)
	if err != nil {
		return Invoice{}, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return Invoice{}, err
	}
	return r.FindByID(ctx, id)
}

// sellerInfo อ่านข้อมูลร้าน + รูปแบบร้าน (ยังไม่ได้ตั้งค่าร้านเลย = ออกเอกสารไม่ได้)
func sellerInfo(ctx context.Context, tx pgx.Tx) (seller, error) {
	var s seller
	err := tx.QueryRow(ctx, `
		SELECT name, address, tax_id, branch, phone, vat_registered, prices_include_vat
		FROM company
		WHERE id = 1`).
		Scan(&s.name, &s.address, &s.taxID, &s.branch, &s.phone, &s.vatRegistered, &s.pricesIncludeVAT)
	if errors.Is(err, pgx.ErrNoRows) {
		return seller{}, ErrSellerTaxIDMissing
	}
	return s, err
}

// nextInvoiceNo เพิ่มเลขของชนิดเอกสาร + ปีนั้นทีละ 1 (ปีแรกเริ่มที่ 1) ใน transaction → กดพร้อมกันเลขก็ไม่ซ้ำ
func nextInvoiceNo(ctx context.Context, tx pgx.Tx, docType, issueDate string) (string, error) {
	var year, number int
	err := tx.QueryRow(ctx, `
		INSERT INTO invoice_numbers (doc_type, year, last_number)
		VALUES ($1, EXTRACT(YEAR FROM $2::date)::int, 1)
		ON CONFLICT (doc_type, year) DO UPDATE SET last_number = invoice_numbers.last_number + 1, updated_at = now()
		RETURNING year, last_number`, docType, issueDate).Scan(&year, &number)
	return invoiceNumber(docPrefix(docType), year, number), err
}

// insertInvoice บันทึกหัวเอกสาร คืน id ใบ
func insertInvoice(ctx context.Context, tx pgx.Tx, req InvoiceRequest, shop seller, docType, invoiceNo string, m money) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO invoices (doc_type, invoice_no, issue_date,
			seller_name, seller_address, seller_tax_id, seller_branch, seller_phone,
			customer_name, customer_address, customer_tax_id, customer_branch,
			note, prices_include_vat, items_total, discount, subtotal, vat, total)
		VALUES ($1, $2, $3::date, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING id`,
		docType, invoiceNo, req.IssueDate,
		shop.name, shop.address, shop.taxID, shop.branch, shop.phone,
		req.CustomerName, req.CustomerAddress, req.CustomerTaxID, req.CustomerBranch,
		req.Note, shop.pricesIncludeVAT && docType == docTaxInvoice, m.itemsTotal, m.discount, m.subtotal, m.vat, m.total).
		Scan(&id)
	return id, err
}

// addItem บันทึกสินค้า 1 บรรทัด: เก็บชื่อ / หน่วย / ต้นทุน ณ วันที่ออก (ต้นทุนไว้คิดกำไรในหน้าภาพรวม)
// สินค้าที่นับสต็อก → ตัด stock + บันทึกประวัติเบิกออก / ไม่นับสต็อก (งานสั่งทำ / บริการ) → ไม่ตัด
func addItem(ctx context.Context, tx pgx.Tx, invoiceID string, lineNo int, item ItemRequest) error {
	var name, unit string
	var trackStock, forSale bool
	var costPrice float64
	err := tx.QueryRow(ctx, `SELECT name, unit, track_stock, for_sale, cost_price FROM products WHERE id = $1`, item.ProductID).
		Scan(&name, &unit, &trackStock, &forSale, &costPrice)
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
		INSERT INTO invoice_items (invoice_id, line_no, product_id, name, unit, quantity, unit_price, amount, stock_cut, cost_price)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		invoiceID, lineNo, item.ProductID, name, unit, item.Quantity, item.UnitPrice,
		lineAmount(item.Quantity, item.UnitPrice), trackStock, costPrice)
	if err != nil {
		return err
	}
	if !trackStock {
		// งานไม่นับสต็อก (รับพิมพ์ / สั่งทำ) → ตัดวัสดุตามใบต้นทุนตอนขาย
		return stock.UseMaterials(ctx, tx, item.ProductID, item.Quantity, invoiceID)
	}
	return cutStock(ctx, tx, invoiceID, item)
}

// cutStock ตัด stock ของสินค้า 1 บรรทัด + บันทึกประวัติเบิกออก
func cutStock(ctx context.Context, tx pgx.Tx, invoiceID string, item ItemRequest) error {
	// stock >= จำนวน → ถ้าของไม่พอ จะไม่มีแถวไหนถูกแก้
	result, err := tx.Exec(ctx, `
		UPDATE products SET stock = stock - $1, updated_at = now()
		WHERE id = $2 AND stock >= $1`, item.Quantity, item.ProductID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrStockNotEnough
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO stock_movements (product_id, type, quantity, invoice_id)
		VALUES ($1, 'out', $2, $3)`, item.ProductID, item.Quantity, invoiceID)
	return err
}

// itemTotals จำนวนรวมของแต่ละสินค้าในใบที่เคยตัด stock (ใบเดียวอาจมีสินค้าเดียวกันหลายบรรทัด)
const itemTotals = `
	SELECT product_id, SUM(quantity) AS quantity
	FROM invoice_items
	WHERE invoice_id = $1 AND stock_cut
	GROUP BY product_id`

// returnStock คืน stock ของสินค้าในใบที่เคยตัดไป + บันทึกประวัติรับเข้า สินค้าละ 1 แถว
func returnStock(ctx context.Context, tx pgx.Tx, invoiceID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE products p SET stock = p.stock + i.quantity, updated_at = now()
		FROM (`+itemTotals+`) i
		WHERE p.id = i.product_id`, invoiceID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO stock_movements (product_id, type, quantity, invoice_id)
		SELECT product_id, 'in', quantity, $1
		FROM (`+itemTotals+`) i`, invoiceID)
	return err
}

// notFoundOrCancelled ยกเลิกไม่ได้ เพราะไม่มีใบนี้ หรือยกเลิกไปแล้ว
func (r *PostgresRepository) notFoundOrCancelled(ctx context.Context, id string) error {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM invoices WHERE id = $1)`, id).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return ErrAlreadyCancelled
}

// items รายการสินค้าในใบ เรียงตามบรรทัด
func (r *PostgresRepository) items(ctx context.Context, invoiceID string) ([]InvoiceItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, product_id, name, unit, quantity, unit_price, amount
		FROM invoice_items
		WHERE invoice_id = $1
		ORDER BY line_no`, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []InvoiceItem{}
	for rows.Next() {
		var item InvoiceItem
		err := rows.Scan(&item.ID, &item.ProductID, &item.Name, &item.Unit, &item.Quantity, &item.UnitPrice, &item.Amount)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
