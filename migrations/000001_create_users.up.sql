-- ตารางผู้ใช้ สำหรับ login ด้วยอีเมล + รหัสผ่าน
-- เก็บรหัสผ่านแบบเข้ารหัส (bcrypt) เท่านั้น ห้ามเก็บรหัสผ่านจริง
CREATE TABLE IF NOT EXISTS users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    first_name    TEXT        NOT NULL,
    last_name     TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
