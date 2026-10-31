-- คำสั่ง SQL ของตาราง company (ข้อมูลร้าน) ไว้ copy ไปใช้ใน DBeaver
-- วิธีใช้: วางเคอร์เซอร์ในคำสั่งที่ต้องการ แล้วกด ⌘ + Enter (รันทีละคำสั่ง)
-- ตารางนี้มีแถวเดียวเสมอ (id = 1) → ไม่มี INSERT / DELETE ใช้ UPDATE อย่างเดียว
-- อัปโหลดโลโก้ใช้ API (PUT /api/v1/company/logo) หรือหน้า Swagger ง่ายกว่า SQL


-- ==================== SELECT (ดูข้อมูล) ====================

-- ดูข้อมูลร้าน (ไม่แสดงตัวไฟล์โลโก้ แสดงแค่ชนิดและขนาด)
SELECT id, name, address, tax_id, branch, phone,
       logo_content_type,
       pg_size_pretty(length(logo)::bigint) AS logo_size,
       created_at, updated_at
FROM company;


-- ==================== UPDATE (แก้ไข) ====================

-- แก้ข้อมูลร้าน
UPDATE company
SET name = 'ร้านตัวอย่าง', address = '123 ถนนสุขุมวิท กรุงเทพฯ 10110',
    tax_id = '0105555555555', branch = 'สำนักงานใหญ่', phone = '02-123-4567',
    updated_at = now()
WHERE id = 1
RETURNING id, name, address, tax_id, branch, phone, updated_at;

-- ลบโลโก้ (กลับไปใช้โลโก้ CostBook)
UPDATE company
SET logo = NULL, logo_content_type = NULL, updated_at = now()
WHERE id = 1
RETURNING id, name, logo_content_type;
