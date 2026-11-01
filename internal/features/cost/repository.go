package cost

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrProductNotFound ใช้บอก handler ว่าไม่เจอสินค้า (handler จะตอบ 404)
var ErrProductNotFound = errors.New("product not found")

// ErrMaterialInvalid วัสดุที่เลือกไม่มีอยู่จริง หรือเลือกตัวเอง (handler จะตอบ 400)
var ErrMaterialInvalid = errors.New("cost material is invalid")

// ErrPresetNotFound ใช้บอก handler ว่าไม่เจอรายการที่ใช้บ่อย (handler จะตอบ 404)
var ErrPresetNotFound = errors.New("cost preset not found")

// Repository คือรายการคำสั่งที่คุยกับ DB
// ทำเป็น interface เพื่อให้ตอน test ใช้ fakeRepository แทน DB จริงได้
type Repository interface {
	Get(ctx context.Context, productID string) (CostSheet, error)
	ResolveMaterials(ctx context.Context, productID string, items []CostItemRequest) ([]CostItemRequest, error)
	Save(ctx context.Context, productID string, req CostSheetRequest, amounts []float64, costPrice float64) (CostSheet, error)
	ListPresets(ctx context.Context) ([]CostPreset, error)
	CreatePreset(ctx context.Context, req PresetRequest) (CostPreset, error)
	UpdatePreset(ctx context.Context, id string, req PresetRequest) (CostPreset, error)
	DeletePreset(ctx context.Context, id string) error
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Get(ctx context.Context, productID string) (CostSheet, error) {
	var sheet CostSheet
	err := r.db.QueryRow(ctx, `
		SELECT id, name, unit, batch_quantity, cost_price, sell_price
		FROM products
		WHERE id = $1`, productID).
		Scan(&sheet.ProductID, &sheet.ProductName, &sheet.Unit, &sheet.BatchQuantity, &sheet.CostPrice, &sheet.SellPrice)
	if errors.Is(err, pgx.ErrNoRows) {
		return CostSheet{}, ErrProductNotFound
	}
	if err != nil {
		return CostSheet{}, err
	}

	sheet.Items, err = r.items(ctx, productID)
	return sheet, err
}

// Save แทนที่รายการต้นทุนทั้งหมดของสินค้า + บันทึกต้นทุน / ราคาขายต่อชิ้นใหม่ ทำใน transaction เดียว
// amounts = จำนวนเงินของแต่ละบรรทัด (handler คำนวณด้วย itemAmounts) เรียงตรงกับ req.Items
func (r *PostgresRepository) Save(ctx context.Context, productID string, req CostSheetRequest, amounts []float64, costPrice float64) (CostSheet, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return CostSheet{}, err
	}
	defer tx.Rollback(ctx) // ถ้า Commit แล้ว คำสั่งนี้ไม่ทำอะไร

	result, err := tx.Exec(ctx, `
		UPDATE products
		SET batch_quantity = $2, cost_price = $3, sell_price = $4, updated_at = now()
		WHERE id = $1`,
		productID, req.BatchQuantity, costPrice, req.SellPrice)
	if err != nil {
		return CostSheet{}, err
	}
	if result.RowsAffected() == 0 {
		return CostSheet{}, ErrProductNotFound
	}

	_, err = tx.Exec(ctx, `DELETE FROM product_costs WHERE product_id = $1`, productID)
	if err != nil {
		return CostSheet{}, err
	}
	for i, item := range req.Items {
		_, err = tx.Exec(ctx, `
			INSERT INTO product_costs (product_id, line_no, type, kind, name, quantity, unit, unit_price, percent, amount, material_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NULLIF($11, '')::uuid)`,
			productID, i+1, item.Type, item.Kind, item.Name, item.Quantity, item.Unit, item.UnitPrice, item.Percent, amounts[i],
			item.MaterialID)
		if err != nil {
			return CostSheet{}, err
		}
	}
	// ต้นทุนของสินค้านี้เปลี่ยน → สินค้าที่ใช้สิ่งนี้เป็นวัสดุต้องคิดต้นทุนใหม่ด้วย
	if err := RefreshDependents(ctx, tx, productID); err != nil {
		return CostSheet{}, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return CostSheet{}, err
	}
	return r.Get(ctx, productID)
}

// ResolveMaterials บรรทัดที่ผูกวัสดุ → ใช้ราคาต่อหน่วย (ต้นทุนของวัสดุ) + หน่วยของวัสดุ ไม่เชื่อค่าที่ส่งมา
// วัสดุไม่มีจริง หรือเป็นสินค้าตัวเอง → ErrMaterialInvalid
func (r *PostgresRepository) ResolveMaterials(ctx context.Context, productID string, items []CostItemRequest) ([]CostItemRequest, error) {
	resolved := make([]CostItemRequest, len(items))
	for i, item := range items {
		resolved[i] = item
		if item.MaterialID == "" {
			continue
		}
		if item.MaterialID == productID {
			return nil, ErrMaterialInvalid
		}
		err := r.db.QueryRow(ctx, `SELECT unit, cost_price FROM products WHERE id = $1`, item.MaterialID).
			Scan(&resolved[i].Unit, &resolved[i].UnitPrice)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMaterialInvalid
		}
		if err != nil {
			return nil, err
		}
	}
	return resolved, nil
}

// items รายการต้นทุนของสินค้า เรียงตามประเภท แล้วตามลำดับที่กรอก
func (r *PostgresRepository) items(ctx context.Context, productID string) ([]CostItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, COALESCE(material_id::text, ''), type, kind, name, quantity, unit, unit_price, percent, amount
		FROM product_costs
		WHERE product_id = $1
		ORDER BY CASE type WHEN 'material' THEN 1 WHEN 'labor' THEN 2 ELSE 3 END, line_no`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []CostItem{} // ไม่ใช้ nil → JSON จะได้ [] แทน null
	for rows.Next() {
		var item CostItem
		err := rows.Scan(&item.ID, &item.MaterialID, &item.Type, &item.Kind, &item.Name, &item.Quantity, &item.Unit,
			&item.UnitPrice, &item.Percent, &item.Amount)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// presetColumns column ที่ใช้อ่านรายการที่ใช้บ่อย เรียงตรงกับ scanPreset
const presetColumns = "id, type, kind, name, unit, unit_price, percent"

func (r *PostgresRepository) ListPresets(ctx context.Context) ([]CostPreset, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+presetColumns+`
		FROM cost_presets
		ORDER BY CASE type WHEN 'material' THEN 1 WHEN 'labor' THEN 2 ELSE 3 END, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	presets := []CostPreset{}
	for rows.Next() {
		p, err := scanPreset(rows)
		if err != nil {
			return nil, err
		}
		presets = append(presets, p)
	}
	return presets, rows.Err()
}

func (r *PostgresRepository) CreatePreset(ctx context.Context, req PresetRequest) (CostPreset, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO cost_presets (type, kind, name, unit, unit_price, percent)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+presetColumns,
		req.Type, req.Kind, req.Name, req.Unit, req.UnitPrice, req.Percent)
	return scanPreset(row)
}

func (r *PostgresRepository) UpdatePreset(ctx context.Context, id string, req PresetRequest) (CostPreset, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE cost_presets
		SET type = $2, kind = $3, name = $4, unit = $5, unit_price = $6, percent = $7, updated_at = now()
		WHERE id = $1
		RETURNING `+presetColumns,
		id, req.Type, req.Kind, req.Name, req.Unit, req.UnitPrice, req.Percent)
	return scanPreset(row)
}

func (r *PostgresRepository) DeletePreset(ctx context.Context, id string) error {
	result, err := r.db.Exec(ctx, `DELETE FROM cost_presets WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrPresetNotFound
	}
	return nil
}

// scanPreset อ่านผลจาก SELECT / RETURNING ใส่ CostPreset
func scanPreset(row pgx.Row) (CostPreset, error) {
	var p CostPreset
	err := row.Scan(&p.ID, &p.Type, &p.Kind, &p.Name, &p.Unit, &p.UnitPrice, &p.Percent)
	if errors.Is(err, pgx.ErrNoRows) {
		return CostPreset{}, ErrPresetNotFound
	}
	return p, err
}
