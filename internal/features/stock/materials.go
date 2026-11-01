package stock

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// UseMaterials ตัดวัสดุที่ใช้ผลิต quantity ชิ้นของ productID ตามใบต้นทุน (บรรทัดที่ผูกวัสดุ) + บันทึกประวัติ
// ใช้ใน transaction เดียวกับการรับเข้า / ออกเอกสารขาย (invoiceID = "" ถ้าไม่ได้มาจากเอกสารขาย)
// วัสดุตัวไหนไม่พอ → *MaterialShortage (ไม่ตัดอะไรเลย เพราะ transaction ถูกยกเลิก) / วัสดุที่ไม่นับสต็อก → ข้าม
func UseMaterials(ctx context.Context, tx pgx.Tx, productID string, quantity int, invoiceID string) error {
	// ล็อกแถววัสดุก่อน กันกดพร้อมกันแล้วตัวเลขเพี้ยน
	_, err := tx.Exec(ctx, `
		SELECT id FROM products
		WHERE id IN (SELECT material_id FROM product_costs WHERE product_id = $1 AND material_id IS NOT NULL)
		FOR UPDATE`, productID)
	if err != nil {
		return err
	}

	rows, err := tx.Query(ctx, `
		SELECT m.id, m.name, m.unit, m.stock, SUM(pc.quantity), p.batch_quantity, p.name
		FROM product_costs pc
		JOIN products m ON m.id = pc.material_id
		JOIN products p ON p.id = pc.product_id
		WHERE pc.product_id = $1 AND m.track_stock
		GROUP BY m.id, m.name, m.unit, m.stock, p.batch_quantity, p.name
		ORDER BY m.name`, productID)
	if err != nil {
		return err
	}
	type use struct {
		id, name, unit string
		need, have     int
	}
	uses := []use{}
	productName := ""
	for rows.Next() {
		var u use
		var perBatch float64
		var batchQuantity int
		if err := rows.Scan(&u.id, &u.name, &u.unit, &u.have, &perBatch, &batchQuantity, &productName); err != nil {
			rows.Close()
			return err
		}
		u.need = materialNeed(perBatch, batchQuantity, quantity)
		if u.need > 0 {
			uses = append(uses, u)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, u := range uses {
		if u.have < u.need {
			return &MaterialShortage{Name: u.name, Unit: u.unit, Need: u.need, Have: u.have}
		}
	}
	note := fmt.Sprintf("ใช้ผลิต %s × %d", productName, quantity)
	for _, u := range uses {
		_, err = tx.Exec(ctx, `UPDATE products SET stock = stock - $2, updated_at = now() WHERE id = $1`, u.id, u.need)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO stock_movements (product_id, type, quantity, note, invoice_id, is_material)
			VALUES ($1, 'out', $2, $3, NULLIF($4, '')::uuid, true)`, u.id, u.need, note, invoiceID)
		if err != nil {
			return err
		}
	}
	return nil
}

// ReturnMaterials ยกเลิกเอกสารขาย → คืนวัสดุที่ตัดไปตอนขาย (งานที่ไม่นับสต็อก) + บันทึกประวัติรับเข้า
func ReturnMaterials(ctx context.Context, tx pgx.Tx, invoiceID string) error {
	const used = `
		SELECT product_id, SUM(quantity) AS quantity
		FROM stock_movements
		WHERE invoice_id = $1 AND is_material AND type = 'out'
		GROUP BY product_id`
	_, err := tx.Exec(ctx, `
		UPDATE products p SET stock = p.stock + u.quantity, updated_at = now()
		FROM (`+used+`) u
		WHERE p.id = u.product_id`, invoiceID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO stock_movements (product_id, type, quantity, note, invoice_id, is_material)
		SELECT product_id, 'in', quantity, 'คืนวัสดุ (ยกเลิกเอกสาร)', $1, true
		FROM (`+used+`) u`, invoiceID)
	return err
}

// WriteShortage วัสดุไม่พอ → ตอบ 409 พร้อมชื่อวัสดุ / ต้องใช้ / มีอยู่ (หน้าเว็บบอกผู้ใช้ได้ว่าต้องรับวัสดุอะไรเข้าก่อน)
// คืน true ถ้า err เป็นวัสดุไม่พอ (handler ต้อง return)
func WriteShortage(c *gin.Context, err error) bool {
	var shortage *MaterialShortage
	if !errors.As(err, &shortage) {
		return false
	}
	c.JSON(http.StatusConflict, gin.H{
		"error":    "material_not_enough",
		"material": shortage.Name,
		"unit":     shortage.Unit,
		"need":     shortage.Need,
		"have":     shortage.Have,
	})
	return true
}
