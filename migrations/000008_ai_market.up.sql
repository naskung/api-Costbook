-- ภาพรวมแบบละเอียด + AI เช็คราคาตลาด

-- 1. ต้นทุนต่อหน่วย ณ วันที่ขาย (ไว้คิดกำไรขั้นต้นของแต่ละเดือน)
-- ต้นทุนสินค้าเปลี่ยนได้ตลอด → ต้องเก็บไว้ในบรรทัดของเอกสารเลย เหมือนชื่อ / ราคา
ALTER TABLE invoice_items
    ADD COLUMN IF NOT EXISTS cost_price NUMERIC(12,2) NOT NULL DEFAULT 0;

-- เอกสารที่ออกก่อนมี column นี้: ใช้ต้นทุนปัจจุบันของสินค้าแทน (ประมาณเอา)
UPDATE invoice_items ii
SET cost_price = p.cost_price
FROM products p
WHERE p.id = ii.product_id AND ii.cost_price = 0;

-- 2. ผลเช็คราคาตลาดด้วย AI (เก็บทุกครั้ง หน้าเว็บใช้ครั้งล่าสุดของแต่ละสินค้า)
-- our_* = ต้นทุน / ราคาขายของร้าน ณ วันที่เช็ค (ไว้เทียบ)
-- market_cost = ต้นทุนต่อหน่วยที่ร้านอื่นในตลาดน่าจะใช้ / 0 = ไม่รู้
-- cost_items = [{"name", "amount", "note"}], sources = [{"title", "url", "price", "kind"}]
CREATE TABLE IF NOT EXISTS market_checks (
    id              UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id      UUID          NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    our_cost_price  NUMERIC(12,2) NOT NULL DEFAULT 0,
    our_sell_price  NUMERIC(12,2) NOT NULL DEFAULT 0,
    retail_low      NUMERIC(12,2) NOT NULL DEFAULT 0,
    retail_high     NUMERIC(12,2) NOT NULL DEFAULT 0,
    retail_typical  NUMERIC(12,2) NOT NULL DEFAULT 0,
    wholesale_low   NUMERIC(12,2) NOT NULL DEFAULT 0,
    wholesale_high  NUMERIC(12,2) NOT NULL DEFAULT 0,
    wholesale_note  TEXT          NOT NULL DEFAULT '',
    market_cost     NUMERIC(12,2) NOT NULL DEFAULT 0,
    suggested_price NUMERIC(12,2) NOT NULL DEFAULT 0,
    confidence      TEXT          NOT NULL CHECK (confidence IN ('low', 'medium', 'high')),
    summary         TEXT          NOT NULL DEFAULT '',
    advice          TEXT          NOT NULL DEFAULT '',
    cost_items      JSONB         NOT NULL DEFAULT '[]',
    sources         JSONB         NOT NULL DEFAULT '[]',
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS market_checks_product_id_idx ON market_checks (product_id, created_at DESC);
