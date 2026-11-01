-- วัสดุ / ส่วนประกอบ = สินค้าที่ไม่ขาย (for_sale = false) อยู่ในสต็อก ใช้ในใบต้นทุนของสินค้าอื่น
ALTER TABLE products
    ADD COLUMN IF NOT EXISTS for_sale BOOLEAN NOT NULL DEFAULT true;

-- บรรทัดต้นทุนที่ผูกกับวัสดุในสต็อก: ราคาต่อหน่วย = ต้นทุนของวัสดุ (อัปเดตเองเมื่อราคาวัสดุเปลี่ยน)
-- รับสินค้าเข้า / ขายงานไม่นับสต็อก → ตัดวัสดุตามบรรทัดเหล่านี้ / ลบวัสดุ → บรรทัดเก็บค่าเดิมไว้ ไม่ผูกแล้ว
ALTER TABLE product_costs
    ADD COLUMN IF NOT EXISTS material_id UUID REFERENCES products (id) ON DELETE SET NULL;

ALTER TABLE product_costs DROP CONSTRAINT IF EXISTS product_costs_material_not_self;
ALTER TABLE product_costs
    ADD CONSTRAINT product_costs_material_not_self CHECK (material_id IS NULL OR material_id <> product_id);

CREATE INDEX IF NOT EXISTS product_costs_material_id_idx ON product_costs (material_id);

-- ประวัติสต็อกที่เป็นการตัด / คืนวัสดุ (ใช้ผลิตสินค้าอื่น) — ยกเลิกเอกสารขายคืนวัสดุจากแถวเหล่านี้
ALTER TABLE stock_movements
    ADD COLUMN IF NOT EXISTS is_material BOOLEAN NOT NULL DEFAULT false;
