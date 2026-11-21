-- ใบเสนอราคา (ผู้ใช้ขอ 2026-11-21) — เสนอราคาให้ลูกค้าก่อนขาย: ไม่ตัดสต็อก ไม่บันทึกรายรับ ไม่นับในภาพรวม
-- แยกตารางจาก invoices เพราะหน้าภาพรวม / รายรับ-รายจ่ายรวมยอดจาก invoices ทุกใบ
-- เก็บข้อมูลผู้ขาย / ลูกค้า / ชื่อสินค้า ณ วันที่ออกไว้ในใบเลย (แบบเดียวกับใบกำกับ)
-- แก้ / ลบไม่ได้ ผิดให้ยกเลิก (status = cancelled) แล้วออกใบใหม่
-- with_vat = ตอนออกร้านจด VAT ไหม (จด = คิด VAT ในใบ)
CREATE TABLE IF NOT EXISTS quotations (
    id                 UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    quotation_no       TEXT          NOT NULL UNIQUE,
    issue_date         DATE          NOT NULL,
    valid_until        DATE          NOT NULL CHECK (valid_until >= issue_date),
    status             TEXT          NOT NULL DEFAULT 'issued' CHECK (status IN ('issued', 'cancelled')),
    seller_name        TEXT          NOT NULL,
    seller_address     TEXT          NOT NULL,
    seller_tax_id      TEXT          NOT NULL,
    seller_branch      TEXT          NOT NULL,
    seller_phone       TEXT          NOT NULL,
    customer_name      TEXT          NOT NULL,
    customer_address   TEXT          NOT NULL DEFAULT '',
    customer_tax_id    TEXT          NOT NULL DEFAULT '',
    customer_branch    TEXT          NOT NULL DEFAULT '',
    note               TEXT          NOT NULL DEFAULT '',
    with_vat           BOOLEAN       NOT NULL,
    prices_include_vat BOOLEAN       NOT NULL DEFAULT FALSE,
    items_total        NUMERIC(12,2) NOT NULL,
    discount           NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (discount >= 0),
    subtotal           NUMERIC(12,2) NOT NULL,
    vat                NUMERIC(12,2) NOT NULL,
    total              NUMERIC(12,2) NOT NULL,
    cancelled_at       TIMESTAMPTZ,
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS quotations_issue_date_idx ON quotations (issue_date);

-- สินค้าในใบเสนอราคา — ลบสินค้าทีหลังได้ (product_id กลายเป็น NULL) ใบยังมีชื่อ / หน่วย / ราคาครบ
CREATE TABLE IF NOT EXISTS quotation_items (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    quotation_id UUID          NOT NULL REFERENCES quotations (id) ON DELETE CASCADE,
    line_no      INT           NOT NULL,
    product_id   UUID          REFERENCES products (id) ON DELETE SET NULL,
    name         TEXT          NOT NULL,
    unit         TEXT          NOT NULL,
    quantity     INT           NOT NULL CHECK (quantity > 0),
    unit_price   NUMERIC(12,2) NOT NULL CHECK (unit_price >= 0),
    amount       NUMERIC(12,2) NOT NULL,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS quotation_items_quotation_id_idx ON quotation_items (quotation_id);

-- เลขล่าสุดของแต่ละปี เช่น ปี 2026 ออกไปแล้ว 5 ใบ → ใบถัดไปได้ QT-2026-0006
CREATE TABLE IF NOT EXISTS quotation_numbers (
    year        INT         PRIMARY KEY,
    last_number INT         NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
