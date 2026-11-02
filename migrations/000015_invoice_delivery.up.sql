-- สถานะการส่งสินค้าของเอกสารขาย (ลูกค้าจ่ายเงินแล้ว แต่อาจยังไม่ได้ของ)
-- pending = รอส่ง, shipped = ส่งแล้ว, received = ลูกค้าได้รับแล้ว
-- ร้านเลือกเปิดใช้ในหน้าตั้งค่า (company.track_delivery) ค่าเริ่มต้นปิด → ร้านเดิมใช้เหมือนเดิม
-- ใบที่มีอยู่ก่อนแล้ว = received (ค่าเริ่มต้นของ column) ถือว่าลูกค้าได้ของไปแล้ว
ALTER TABLE company ADD COLUMN IF NOT EXISTS track_delivery BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS delivery_status TEXT NOT NULL DEFAULT 'received';
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_delivery_status_check;
ALTER TABLE invoices ADD CONSTRAINT invoices_delivery_status_check
    CHECK (delivery_status IN ('pending', 'shipped', 'received'));
