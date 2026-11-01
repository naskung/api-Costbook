package cost

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DB คำสั่ง SQL ที่ใช้ได้ทั้ง *pgxpool.Pool และ pgx.Tx (feature อื่นเรียก RefreshDependents ใน transaction ของตัวเองได้)
type DB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// RefreshDependents ต้นทุนของ materialID เปลี่ยน → คิดต้นทุนใหม่ให้สินค้าทุกตัวที่ใช้เป็นวัสดุในใบต้นทุน
// สินค้าที่คิดใหม่เป็นวัสดุของตัวอื่นอีก → คิดต่อไปเรื่อย ๆ (จำไว้ว่าคิดตัวไหนแล้ว กันวนไม่จบ)
// ใช้ตอน: บันทึกใบต้นทุน / รับเข้าแล้วต้นทุนเฉลี่ยเปลี่ยน / แก้ต้นทุนสินค้าเอง
func RefreshDependents(ctx context.Context, db DB, materialID string) error {
	queue := []string{materialID}
	done := map[string]bool{materialID: true}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		dependents, err := dependentsOf(ctx, db, current)
		if err != nil {
			return err
		}
		for _, productID := range dependents {
			if done[productID] {
				continue
			}
			done[productID] = true
			if err := recalculate(ctx, db, productID); err != nil {
				return err
			}
			queue = append(queue, productID)
		}
	}
	return nil
}

// dependentsOf สินค้าที่ใช้ materialID ในใบต้นทุน
func dependentsOf(ctx context.Context, db DB, materialID string) ([]string, error) {
	rows, err := db.Query(ctx, `SELECT DISTINCT product_id FROM product_costs WHERE material_id = $1`, materialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// recalculate ดึงราคาวัสดุล่าสุดใส่บรรทัดที่ผูกวัสดุ แล้วคิดเงินทุกบรรทัด + ต้นทุนต่อชิ้นใหม่ (กฎเดียวกับตอนบันทึกใบต้นทุน)
func recalculate(ctx context.Context, db DB, productID string) error {
	_, err := db.Exec(ctx, `
		UPDATE product_costs pc
		SET unit_price = m.cost_price, unit = m.unit, updated_at = now()
		FROM products m
		WHERE pc.product_id = $1 AND pc.material_id = m.id`, productID)
	if err != nil {
		return err
	}

	rows, err := db.Query(ctx, `
		SELECT id, type, kind, quantity, unit_price, percent
		FROM product_costs
		WHERE product_id = $1
		ORDER BY line_no`, productID)
	if err != nil {
		return err
	}
	ids := []string{}
	items := []CostItemRequest{}
	for rows.Next() {
		var id string
		var item CostItemRequest
		if err := rows.Scan(&id, &item.Type, &item.Kind, &item.Quantity, &item.UnitPrice, &item.Percent); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
		items = append(items, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	var batchQuantity int
	if err := db.QueryRow(ctx, `SELECT batch_quantity FROM products WHERE id = $1`, productID).Scan(&batchQuantity); err != nil {
		return err
	}
	amounts := itemAmounts(items)
	for i, id := range ids {
		if _, err := db.Exec(ctx, `UPDATE product_costs SET amount = $2 WHERE id = $1`, id, amounts[i]); err != nil {
			return err
		}
	}
	_, err = db.Exec(ctx, `UPDATE products SET cost_price = $2, updated_at = now() WHERE id = $1`,
		productID, costPrice(items, amounts, batchQuantity))
	return err
}
