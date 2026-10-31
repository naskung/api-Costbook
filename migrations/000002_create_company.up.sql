-- ข้อมูลร้าน (ผู้ขาย) + โลโก้ ใช้บนหน้าเว็บและใบกำกับภาษี
-- 1 ระบบ = 1 ร้าน → มีได้แถวเดียว (id = 1 เสมอ)
-- ร้านใหม่ = ติดตั้งระบบแยก แล้วแก้ข้อมูลผ่าน API (หน้าเว็บไม่ต้องแก้)
CREATE TABLE IF NOT EXISTS company (
    id                INT         PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    name              TEXT        NOT NULL DEFAULT '',
    address           TEXT        NOT NULL DEFAULT '',
    tax_id            TEXT        NOT NULL DEFAULT '',
    branch            TEXT        NOT NULL DEFAULT '',
    phone             TEXT        NOT NULL DEFAULT '',
    logo              BYTEA,
    logo_content_type TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- สร้างแถวว่างไว้เลย API จะได้ UPDATE อย่างเดียว ไม่ต้องเช็คว่ามีแถวหรือยัง
INSERT INTO company (id) VALUES (1) ON CONFLICT (id) DO NOTHING;
