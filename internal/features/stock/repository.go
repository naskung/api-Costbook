package stock

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/naskung/api-Costbook/internal/features/cost"
)

// ErrProductNotFound ใช้บอก handler ว่าไม่เจอสินค้า (handler จะตอบ 404)
var ErrProductNotFound = errors.New("product not found")

// ErrNotEnough เบิกออกมากกว่าที่มีอยู่ (handler จะตอบ 409)
var ErrNotEnough = errors.New("stock not enough")

// ErrNotTracked สินค้านี้ตั้งไว้ว่าไม่นับสต็อก (งานสั่งทำ / ค่าบริการ) (handler จะตอบ 409)
var ErrNotTracked = errors.New("product does not track stock")

// Repository คือรายการคำสั่งที่คุยกับ DB
// ทำเป็น interface เพื่อให้ตอน test ใช้ fakeRepository แทน DB จริงได้
type Repository interface {
	List(ctx context.Context, onlyLow bool) ([]StockItem, error)
	Movements(ctx context.Context, productID string) ([]Movement, error)
	AddMovement(ctx context.Context, movementType string, req MovementRequest) (Movement, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// List จำนวนคงเหลือของทุกสินค้าที่นับสต็อก (onlyLow = true → เฉพาะที่ใกล้หมด)
func (r *PostgresRepository) List(ctx context.Context, onlyLow bool) ([]StockItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, unit, stock, min_stock, stock <= min_stock
		FROM products
		WHERE track_stock AND ($1 = false OR stock <= min_stock)
		ORDER BY stock <= min_stock DESC, name`, onlyLow)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []StockItem{} // ไม่ใช้ nil → JSON จะได้ [] แทน null
	for rows.Next() {
		var item StockItem
		err := rows.Scan(&item.ProductID, &item.Name, &item.Unit, &item.Stock, &item.MinStock, &item.IsLow)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// saleShare สัดส่วนยอดก่อน VAT หลังหักส่วนลดของบิล ต่อยอดรวมสินค้า → ยอดบรรทัด × saleShare = ยอดขายสุทธิของบรรทัด
// (ต้องตรงกับ netAmount ของหน้าภาพรวม — dashboard/repository.go)
const saleShare = `CASE WHEN i.items_total > 0 THEN i.subtotal / i.items_total ELSE 0 END`

// movementSelect อ่านประวัติพร้อมชื่อสินค้าและเลขที่ใบกำกับ เรียงตรงกับ scanMovement
// s = ยอดของสินค้านี้ในบิล (จำนวน / ยอดขายสุทธิ / ต้นทุน ณ วันที่ขาย) เฉพาะแถวที่ขายตามบิลที่ยังไม่ยกเลิก ไว้คิดกำไร
const movementSelect = `
	SELECT m.id, m.product_id, p.name, p.unit, m.type, m.quantity, m.note, COALESCE(i.invoice_no, ''),
		m.unit_cost, m.supplier, m.is_material, m.created_at,
		COALESCE(s.quantity, 0), COALESCE(s.amount * ` + saleShare + `, 0), COALESCE(s.cost, 0)
	FROM stock_movements m
	JOIN products p ON p.id = m.product_id
	LEFT JOIN invoices i ON i.id = m.invoice_id
	LEFT JOIN LATERAL (
		SELECT SUM(ii.quantity) AS quantity, SUM(ii.amount) AS amount, SUM(ii.quantity * ii.cost_price) AS cost
		FROM invoice_items ii
		WHERE ii.invoice_id = m.invoice_id AND ii.product_id = m.product_id
	) s ON m.type = 'out' AND NOT m.is_material AND i.status = 'issued'`

// Movements ประวัติล่าสุด 200 รายการ (productID ว่าง = ทุกสินค้า)
func (r *PostgresRepository) Movements(ctx context.Context, productID string) ([]Movement, error) {
	rows, err := r.db.Query(ctx, movementSelect+`
		WHERE $1 = '' OR m.product_id::text = $1
		ORDER BY m.created_at DESC, p.name
		LIMIT 200`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	movements := []Movement{}
	for rows.Next() {
		m, err := scanMovement(rows)
		if err != nil {
			return nil, err
		}
		movements = append(movements, m)
	}
	return movements, rows.Err()
}

// AddMovement รับเข้า (in) / เบิกออก (out): แก้จำนวนคงเหลือ (+ ต้นทุนเฉลี่ย) + บันทึกประวัติ ทำใน transaction เดียว
// รับเข้า = ผลิตเสร็จ → ตัดวัสดุตามใบต้นทุน (ยกเว้น skip_materials) / ต้นทุนเปลี่ยน → คิดต้นทุนสินค้าที่ใช้สิ่งนี้เป็นวัสดุใหม่
func (r *PostgresRepository) AddMovement(ctx context.Context, movementType string, req MovementRequest) (Movement, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Movement{}, err
	}
	defer tx.Rollback(ctx) // ถ้า Commit แล้ว คำสั่งนี้ไม่ทำอะไร

	stock, oldCost, err := lockProduct(ctx, tx, req.ProductID)
	if err != nil {
		return Movement{}, err
	}
	newCost := oldCost

	if movementType == "out" {
		if stock < req.Quantity {
			return Movement{}, ErrNotEnough
		}
		stock -= req.Quantity
	} else {
		if req.UnitCost > 0 {
			newCost = averageCost(stock, oldCost, req.Quantity, req.UnitCost)
		}
		stock += req.Quantity
		if !req.SkipMaterials {
			if err := UseMaterials(ctx, tx, req.ProductID, req.Quantity, ""); err != nil {
				return Movement{}, err
			}
		}
	}

	_, err = tx.Exec(ctx, `UPDATE products SET stock = $2, cost_price = $3, updated_at = now() WHERE id = $1`,
		req.ProductID, stock, newCost)
	if err != nil {
		return Movement{}, err
	}
	if newCost != oldCost {
		if err := cost.RefreshDependents(ctx, tx, req.ProductID); err != nil {
			return Movement{}, err
		}
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO stock_movements (product_id, type, quantity, note, unit_cost, supplier)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`, req.ProductID, movementType, req.Quantity, req.Note, req.UnitCost, req.Supplier).Scan(&id)
	if err != nil {
		return Movement{}, err
	}

	movement, err := scanMovement(tx.QueryRow(ctx, movementSelect+` WHERE m.id = $1`, id))
	if err != nil {
		return Movement{}, err
	}
	return movement, tx.Commit(ctx)
}

// lockProduct อ่านจำนวนคงเหลือ + ต้นทุนของสินค้า แล้วล็อกแถวไว้จนจบ transaction (กันกดพร้อมกันแล้วตัวเลขเพี้ยน)
func lockProduct(ctx context.Context, tx pgx.Tx, productID string) (int, float64, error) {
	var stock int
	var cost float64
	var trackStock bool
	err := tx.QueryRow(ctx, `SELECT stock, cost_price, track_stock FROM products WHERE id = $1 FOR UPDATE`, productID).
		Scan(&stock, &cost, &trackStock)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, ErrProductNotFound
	}
	if err != nil {
		return 0, 0, err
	}
	if !trackStock {
		return 0, 0, ErrNotTracked
	}
	return stock, cost, nil
}

// scanMovement อ่านผลจาก movementSelect ใส่ Movement
func scanMovement(row pgx.Row) (Movement, error) {
	var m Movement
	var lineQuantity int
	var lineNet, lineCost float64
	err := row.Scan(&m.ID, &m.ProductID, &m.ProductName, &m.Unit, &m.Type, &m.Quantity, &m.Note, &m.InvoiceNo,
		&m.UnitCost, &m.Supplier, &m.IsMaterial, &m.CreatedAt, &lineQuantity, &lineNet, &lineCost)
	return withSaleProfit(m, lineQuantity, lineNet, lineCost), err
}
