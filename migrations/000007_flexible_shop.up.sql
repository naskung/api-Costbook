-- ทำให้ระบบใช้ได้กับร้านทุกรูปแบบ: ร้านผลิตเอง / ร้านซื้อมาขายไป / ร้านจด VAT หรือไม่จด
-- เพิ่ม column ใหม่ทั้งหมดมีค่าเริ่มต้น → ข้อมูลเดิมยังใช้ได้เหมือนเดิม

-- 1. รูปแบบร้าน (ตั้งในหน้าตั้งค่า)
-- vat_registered = false → ออกใบเสร็จรับเงิน ไม่คิด VAT / prices_include_vat = true → ราคาขายรวม VAT แล้ว
ALTER TABLE company
    ADD COLUMN IF NOT EXISTS vat_registered     BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS prices_include_vat BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS use_cost_page      BOOLEAN NOT NULL DEFAULT TRUE;

-- 2. ข้อมูลสินค้ากลาง + ช่องข้อมูลเพิ่มเติมที่ร้านตั้งเอง
-- track_stock = false → งานสั่งทำ / ค่าบริการ ไม่นับสต็อก
-- custom_fields = ค่าของช่องเพิ่มเติม {"<id ช่อง>": "ค่า"} (นิยามช่องอยู่ในตาราง product_fields)
ALTER TABLE products
    ADD COLUMN IF NOT EXISTS code          TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS category      TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS description   TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS track_stock   BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS custom_fields JSONB   NOT NULL DEFAULT '{}';

-- รหัส / บาร์โค้ดห้ามซ้ำ (ไม่กรอกได้หลายตัว)
CREATE UNIQUE INDEX IF NOT EXISTS products_code_key ON products (code) WHERE code <> '';

-- นิยามช่องข้อมูลเพิ่มเติม type: text / number / date / select (select ใช้ options)
CREATE TABLE IF NOT EXISTS product_fields (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT        NOT NULL,
    type         TEXT        NOT NULL CHECK (type IN ('text', 'number', 'date', 'select')),
    options      TEXT[]      NOT NULL DEFAULT '{}',
    show_in_list BOOLEAN     NOT NULL DEFAULT FALSE,
    sort_order   INT         NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 3. ต้นทุนแต่ละบรรทัด kind: fixed = จำนวน × ราคาต่อหน่วย, percent = % ของรวมบรรทัด fixed (เช่น เผื่อเสีย 10%)
-- amount = ผลลัพธ์ที่ API คำนวณแล้ว
ALTER TABLE product_costs
    ADD COLUMN IF NOT EXISTS kind       TEXT          NOT NULL DEFAULT 'fixed' CHECK (kind IN ('fixed', 'percent')),
    ADD COLUMN IF NOT EXISTS quantity   NUMERIC(14,3) NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS unit       TEXT          NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS unit_price NUMERIC(14,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS percent    NUMERIC(7,2)  NOT NULL DEFAULT 0;

-- รายการเดิมที่มีแค่ amount → ให้เป็น 1 × amount
UPDATE product_costs SET unit_price = amount WHERE kind = 'fixed' AND quantity = 1 AND unit_price = 0;

-- รายการต้นทุนที่ใช้บ่อย (เช่น เส้น PLA กรัมละ 0.45, ค่าเครื่องชั่วโมงละ 6) ไว้เลือกซ้ำ
CREATE TABLE IF NOT EXISTS cost_presets (
    id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    type       TEXT          NOT NULL CHECK (type IN ('material', 'labor', 'overhead')),
    kind       TEXT          NOT NULL DEFAULT 'fixed' CHECK (kind IN ('fixed', 'percent')),
    name       TEXT          NOT NULL,
    unit       TEXT          NOT NULL DEFAULT '',
    unit_price NUMERIC(14,4) NOT NULL DEFAULT 0,
    percent    NUMERIC(7,2)  NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ   NOT NULL DEFAULT now()
);

-- 4. รับเข้าพร้อมราคาทุน (ร้านซื้อมาขายไป) → คิดต้นทุนเฉลี่ยใหม่ / ผู้ขายส่ง
ALTER TABLE stock_movements
    ADD COLUMN IF NOT EXISTS unit_cost NUMERIC(14,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS supplier  TEXT          NOT NULL DEFAULT '';

-- 5. เอกสารขายหลายแบบ doc_type: tax_invoice = ใบกำกับภาษี, receipt = ใบเสร็จรับเงิน (ไม่มี VAT)
-- items_total = รวมราคาสินค้า, discount = ส่วนลดทั้งบิล, subtotal = มูลค่าก่อน VAT
ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS doc_type           TEXT          NOT NULL DEFAULT 'tax_invoice' CHECK (doc_type IN ('tax_invoice', 'receipt')),
    ADD COLUMN IF NOT EXISTS prices_include_vat BOOLEAN       NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS items_total        NUMERIC(12,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS discount           NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (discount >= 0);

UPDATE invoices SET items_total = subtotal WHERE items_total = 0;

-- stock_cut = ตอนออกใบได้ตัดสต็อกไหม (สินค้าไม่นับสต็อกไม่ตัด) → ยกเลิกใบคืนสต็อกเฉพาะที่เคยตัด
ALTER TABLE invoice_items
    ADD COLUMN IF NOT EXISTS stock_cut BOOLEAN NOT NULL DEFAULT TRUE;

-- รันเลขแยกตามชนิดเอกสาร: INV-2026-0001 (ใบกำกับ), RC-2026-0001 (ใบเสร็จ)
ALTER TABLE invoice_numbers
    ADD COLUMN IF NOT EXISTS doc_type TEXT NOT NULL DEFAULT 'tax_invoice';
ALTER TABLE invoice_numbers DROP CONSTRAINT IF EXISTS invoice_numbers_pkey;
ALTER TABLE invoice_numbers ADD PRIMARY KEY (doc_type, year);
