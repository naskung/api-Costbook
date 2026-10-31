-- ใบกำกับภาษี + รายการสินค้าในใบ + ตัวรันเลขที่ใบ
-- เก็บข้อมูลผู้ขาย / ผู้ซื้อ / ชื่อสินค้า ณ วันที่ออกไว้ในใบเลย → แก้ข้อมูลร้านหรือสินค้าทีหลัง ใบเก่าไม่เปลี่ยน
-- ใบกำกับแก้ / ลบไม่ได้ ถ้าผิดให้ยกเลิก (status = cancelled) แล้วออกใบใหม่
CREATE TABLE IF NOT EXISTS invoices (
    id               UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_no       TEXT          NOT NULL UNIQUE,
    issue_date       DATE          NOT NULL,
    status           TEXT          NOT NULL DEFAULT 'issued' CHECK (status IN ('issued', 'cancelled')),
    seller_name      TEXT          NOT NULL,
    seller_address   TEXT          NOT NULL,
    seller_tax_id    TEXT          NOT NULL,
    seller_branch    TEXT          NOT NULL,
    seller_phone     TEXT          NOT NULL,
    customer_name    TEXT          NOT NULL,
    customer_address TEXT          NOT NULL DEFAULT '',
    customer_tax_id  TEXT          NOT NULL DEFAULT '',
    customer_branch  TEXT          NOT NULL DEFAULT '',
    note             TEXT          NOT NULL DEFAULT '',
    subtotal         NUMERIC(12,2) NOT NULL,
    vat              NUMERIC(12,2) NOT NULL,
    total            NUMERIC(12,2) NOT NULL,
    cancelled_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS invoices_issue_date_idx ON invoices (issue_date);

-- สินค้าที่เคยอยู่ในใบกำกับ ลบไม่ได้ (ON DELETE RESTRICT)
-- line_no = ลำดับบรรทัดในใบ 1, 2, 3 ... (บันทึกพร้อมกันทั้งใบ created_at จะเท่ากันหมด เรียงด้วย created_at ไม่ได้)
CREATE TABLE IF NOT EXISTS invoice_items (
    id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID          NOT NULL REFERENCES invoices (id) ON DELETE CASCADE,
    line_no    INT           NOT NULL,
    product_id UUID          NOT NULL REFERENCES products (id) ON DELETE RESTRICT,
    name       TEXT          NOT NULL,
    unit       TEXT          NOT NULL,
    quantity   INT           NOT NULL CHECK (quantity > 0),
    unit_price NUMERIC(12,2) NOT NULL CHECK (unit_price >= 0),
    amount     NUMERIC(12,2) NOT NULL,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS invoice_items_invoice_id_idx ON invoice_items (invoice_id);
CREATE INDEX IF NOT EXISTS invoice_items_product_id_idx ON invoice_items (product_id);

-- เลขล่าสุดของแต่ละปี เช่น ปี 2026 ออกไปแล้ว 5 ใบ → last_number = 5 ใบถัดไปได้ INV-2026-0006
CREATE TABLE IF NOT EXISTS invoice_numbers (
    year        INT         PRIMARY KEY,
    last_number INT         NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
