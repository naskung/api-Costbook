-- ย้อนกลับ: เอาชื่อหมวดกลับไปเก็บเป็นข้อความในสินค้า แล้วลบตารางหมวด
ALTER TABLE products ADD COLUMN IF NOT EXISTS category TEXT NOT NULL DEFAULT '';

UPDATE products p
SET category = c.name
FROM product_categories c
WHERE c.id = p.category_id;

DROP INDEX IF EXISTS products_category_id_idx;
ALTER TABLE products DROP COLUMN IF EXISTS category_id;
DROP TABLE IF EXISTS product_categories;
