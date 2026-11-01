-- role ของผู้ใช้ (เผื่อแบ่งสิทธิ์ว่า role ไหนเห็นเมนูไหน / ทำอะไรได้) + ปิดบัญชีได้แทนการลบ + รูปโปรไฟล์
-- owner = เจ้าของร้าน (ทำได้ทุกอย่าง จัดการผู้ใช้ได้), manager = ผู้จัดการ, staff = พนักงาน
-- ผู้ใช้ที่มีอยู่ก่อนแล้วเป็น owner ทั้งหมด (ค่าเริ่มต้นของ column)
ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'owner';
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('owner', 'manager', 'staff'));
ALTER TABLE users ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar BYTEA;
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_content_type TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_version BIGINT NOT NULL DEFAULT 0;
