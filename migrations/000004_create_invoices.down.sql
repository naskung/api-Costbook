-- ย้อนกลับ: ลบตารางใบกำกับภาษีทั้งหมด (ต้องลบ stock_movements / transactions ก่อน เพราะผูกกับใบกำกับ)
DROP TABLE IF EXISTS invoice_numbers;
DROP TABLE IF EXISTS invoice_items;
DROP TABLE IF EXISTS invoices;
