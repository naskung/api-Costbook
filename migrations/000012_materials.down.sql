-- ย้อนกลับ: ไม่แยกวัสดุ / ไม่ผูกใบต้นทุนกับวัสดุ
ALTER TABLE stock_movements DROP COLUMN IF EXISTS is_material;
DROP INDEX IF EXISTS product_costs_material_id_idx;
ALTER TABLE product_costs DROP CONSTRAINT IF EXISTS product_costs_material_not_self;
ALTER TABLE product_costs DROP COLUMN IF EXISTS material_id;
ALTER TABLE products DROP COLUMN IF EXISTS for_sale;
