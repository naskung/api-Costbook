-- ย้อนกลับ: ลบ key ของ AI ที่ร้านใส่ไว้
ALTER TABLE company DROP COLUMN IF EXISTS ai_key;
