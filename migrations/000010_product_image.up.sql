-- รูปสินค้า 1 รูป เก็บใน DB แบบเดียวกับโลโก้ร้าน (PNG / JPG / WebP ไม่เกิน 1 MB)
-- image_version = เวลาที่เปลี่ยนรูปล่าสุด (มิลลิวินาที) ใส่ท้าย URL รูป → เปลี่ยนรูปแล้ว browser โหลดใหม่ / 0 = ไม่มีรูป
ALTER TABLE products
    ADD COLUMN IF NOT EXISTS image              BYTEA,
    ADD COLUMN IF NOT EXISTS image_content_type TEXT   NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS image_version      BIGINT NOT NULL DEFAULT 0;
