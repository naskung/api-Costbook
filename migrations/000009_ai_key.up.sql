-- key ของ Claude ที่ร้านใส่เองในหน้าตั้งค่า (เข้ารหัสแล้ว ห้ามเก็บเป็นข้อความตรง ๆ) / ว่าง = ใช้ AI ฟรี
ALTER TABLE company
    ADD COLUMN IF NOT EXISTS ai_key TEXT NOT NULL DEFAULT '';
