-- คำสั่ง SQL ของตาราง users ไว้ copy ไปใช้ใน DBeaver
-- วิธีใช้: วางเคอร์เซอร์ในคำสั่งที่ต้องการ แล้วกด ⌘ + Enter (รันทีละคำสั่ง)
-- ห้ามกด Execute script (⌥ + X) เพราะจะรันทุกคำสั่งในไฟล์ รวมถึง DELETE
-- เปลี่ยนอีเมล / ชื่อ / รหัสผ่านในตัวอย่างเป็นของจริงก่อนรัน


-- ==================== SELECT (ดูข้อมูล) ====================

-- ดูผู้ใช้ทั้งหมด (ไม่แสดง password_hash)
SELECT id, email, first_name, last_name, created_at, updated_at
FROM users
ORDER BY created_at DESC;

-- หาผู้ใช้จากอีเมล
SELECT id, email, first_name, last_name, created_at, updated_at
FROM users
WHERE email = 'somchai@example.com';

-- ค้นหาจากชื่อหรือนามสกุล (พิมพ์แค่บางส่วนก็เจอ)
SELECT id, email, first_name, last_name
FROM users
WHERE first_name ILIKE '%สม%' OR last_name ILIKE '%สม%';

-- นับจำนวนผู้ใช้
SELECT count(*) AS total_users
FROM users;


-- ==================== INSERT (เพิ่มผู้ใช้) ====================

-- รันครั้งแรกครั้งเดียว: เปิดส่วนเสริม pgcrypto เพื่อใช้ crypt() เข้ารหัสรหัสผ่าน
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- เพิ่มผู้ใช้ 1 คน — crypt(..., gen_salt('bf', 10)) = เข้ารหัสแบบ bcrypt เหมือนที่ API ใช้ จึง login ผ่าน API ได้
-- อีเมลใช้ตัวพิมพ์เล็กทั้งหมด
INSERT INTO users (email, password_hash, first_name, last_name)
VALUES ('somchai@example.com', crypt('1234', gen_salt('bf', 10)), 'สมชาย', 'ใจดี')
RETURNING id, email, first_name, last_name, created_at;

-- เช็คว่ารหัสผ่านถูกไหม (password_ok = true แปลว่าถูก)
SELECT email, password_hash = crypt('1234', password_hash) AS password_ok
FROM users
WHERE email = 'somchai@example.com';


-- ==================== UPDATE (แก้ไข) ====================
-- ต้องมี WHERE ทุกครั้ง ถ้าลืม จะแก้ผู้ใช้ทุกคน

-- แก้ชื่อ-นามสกุล
UPDATE users
SET first_name = 'สมชาย', last_name = 'ใจงาม', updated_at = now()
WHERE email = 'somchai@example.com'
RETURNING id, email, first_name, last_name, updated_at;

-- เปลี่ยนรหัสผ่าน
UPDATE users
SET password_hash = crypt('รหัสผ่านใหม่', gen_salt('bf', 10)), updated_at = now()
WHERE email = 'somchai@example.com'
RETURNING id, email, updated_at;


-- ==================== DELETE (ลบ) ====================
-- ต้องมี WHERE ทุกครั้ง ถ้าลืม จะลบผู้ใช้ทุกคน และลบแล้วกู้คืนไม่ได้

-- ลบผู้ใช้ 1 คน
DELETE FROM users
WHERE email = 'somchai@example.com'
RETURNING id, email;
