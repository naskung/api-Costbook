-- หมวดหมู่สินค้าแยกเป็นตาราง (เดิมเป็นข้อความในสินค้าแต่ละตัว) → แก้ชื่อหมวดครั้งเดียว สินค้าทุกตัวเปลี่ยนตาม
CREATE TABLE IF NOT EXISTS product_categories (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ชื่อหมวดห้ามซ้ำ (ไม่สนตัวพิมพ์เล็ก / ใหญ่ และช่องว่างหน้า-หลัง)
CREATE UNIQUE INDEX IF NOT EXISTS product_categories_name_key ON product_categories (lower(btrim(name)));

-- สินค้าชี้ไปที่หมวด / ลบหมวด → สินค้าในหมวดนั้นกลายเป็นไม่มีหมวด
ALTER TABLE products
    ADD COLUMN IF NOT EXISTS category_id UUID REFERENCES product_categories (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS products_category_id_idx ON products (category_id);

-- ย้ายหมวดเดิมที่พิมพ์ไว้ในสินค้า → แถวในตารางหมวด แล้วลบ column เดิม
-- ทำเฉพาะตอนที่ยังมี column category อยู่ (รัน migration ซ้ำได้ไม่พัง)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'products' AND column_name = 'category') THEN
        -- ชื่อเดียวกันแต่พิมพ์ต่างกัน (เช่น 3D / 3d) → รวมเป็นหมวดเดียว ใช้ชื่อที่ใช้บ่อยสุด (เท่ากัน = ชื่อที่ใช้ก่อน)
        INSERT INTO product_categories (name)
        SELECT DISTINCT ON (lower(name)) name
        FROM (
            SELECT btrim(category) AS name, COUNT(*) AS uses, MIN(created_at) AS first_used
            FROM products
            WHERE btrim(category) <> ''
            GROUP BY btrim(category)
        ) spellings
        ORDER BY lower(name), uses DESC, first_used, name COLLATE "C"
        ON CONFLICT DO NOTHING;

        UPDATE products p
        SET category_id = c.id
        FROM product_categories c
        WHERE lower(btrim(c.name)) = lower(btrim(p.category));

        ALTER TABLE products DROP COLUMN category;
    END IF;
END $$;
