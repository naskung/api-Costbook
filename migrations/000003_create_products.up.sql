-- สินค้า + รายการต้นทุนของสินค้า
-- cost_price (ต้นทุนต่อชิ้น) คำนวณจาก product_costs ทุกครั้งที่บันทึกหน้าคำนวณต้นทุน
-- stock (จำนวนคงเหลือ) แก้ผ่านการรับเข้า / เบิกออก / ใบกำกับเท่านั้น ไม่แก้ตรง ๆ
CREATE TABLE IF NOT EXISTS products (
    id             UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    name           TEXT          NOT NULL,
    unit           TEXT          NOT NULL DEFAULT '',
    cost_price     NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (cost_price >= 0),
    sell_price     NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (sell_price >= 0),
    stock          INT           NOT NULL DEFAULT 0 CHECK (stock >= 0),
    min_stock      INT           NOT NULL DEFAULT 0 CHECK (min_stock >= 0),
    batch_quantity INT           NOT NULL DEFAULT 1 CHECK (batch_quantity > 0),
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT now()
);

-- รายการต้นทุนของสินค้า 1 ตัว (ลบสินค้า → รายการต้นทุนหายไปด้วย)
-- type: material = วัตถุดิบ, labor = ค่าแรง, overhead = ค่าใช้จ่ายอื่น
-- line_no = ลำดับที่ผู้ใช้กรอก (บันทึกพร้อมกันทั้งใบ created_at จะเท่ากันหมด เรียงด้วย created_at ไม่ได้)
CREATE TABLE IF NOT EXISTS product_costs (
    id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID          NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    line_no    INT           NOT NULL,
    type       TEXT          NOT NULL CHECK (type IN ('material', 'labor', 'overhead')),
    name       TEXT          NOT NULL,
    amount     NUMERIC(12,2) NOT NULL CHECK (amount >= 0),
    created_at TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS product_costs_product_id_idx ON product_costs (product_id);
