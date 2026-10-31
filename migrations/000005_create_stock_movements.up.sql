-- ประวัติรับสินค้าเข้า / เบิกออก
-- type: in = รับเข้า, out = เบิกออก / invoice_id มีค่า = เกิดจากใบกำกับ (ออกใบ = out, ยกเลิกใบ = in)
-- ลบสินค้า → ประวัติของสินค้านั้นหายไปด้วย
CREATE TABLE IF NOT EXISTS stock_movements (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID        NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    type       TEXT        NOT NULL CHECK (type IN ('in', 'out')),
    quantity   INT         NOT NULL CHECK (quantity > 0),
    note       TEXT        NOT NULL DEFAULT '',
    invoice_id UUID        REFERENCES invoices (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS stock_movements_product_id_idx ON stock_movements (product_id, created_at);
