-- ลำดับช่องในฟอร์มเพิ่ม / แก้สินค้า — ร้านลากย้ายตำแหน่งได้ที่หน้าตั้งค่า (ผู้ใช้ขอ 2026-11-16)
-- ช่องหลักของระบบ (ย้ายได้ ลบไม่ได้) เก็บเลขลำดับที่นี่ / ช่องที่ร้านเพิ่มเองใช้ product_fields.sort_order
-- ทั้งสองใช้เลขชุดเดียวกัน → เอามาเรียงรวมกันได้ (น้อย = อยู่บน)
CREATE TABLE IF NOT EXISTS product_form_blocks (
    key        TEXT        PRIMARY KEY CHECK (key IN ('image', 'name', 'code_category', 'unit', 'prices', 'sale_stock', 'description')),
    sort_order INT         NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ลำดับเริ่มต้น = แบบเดิมของฟอร์ม
INSERT INTO product_form_blocks (key, sort_order) VALUES
    ('image', 1), ('name', 2), ('code_category', 3), ('unit', 4), ('prices', 5), ('sale_stock', 6), ('description', 7)
ON CONFLICT (key) DO NOTHING;

-- ช่องที่ร้านเพิ่มไว้แล้ว อยู่ต่อท้ายช่องหลักเหมือนเดิม (8, 9, 10, ... ตามลำดับเดิม)
UPDATE product_fields f
SET sort_order = 7 + ordered.position
FROM (SELECT id, ROW_NUMBER() OVER (ORDER BY sort_order, created_at) AS position FROM product_fields) ordered
WHERE f.id = ordered.id;
