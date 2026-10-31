-- ย้อนกลับ: ลบรูปสินค้าทั้งหมด
ALTER TABLE products
    DROP COLUMN IF EXISTS image,
    DROP COLUMN IF EXISTS image_content_type,
    DROP COLUMN IF EXISTS image_version;
