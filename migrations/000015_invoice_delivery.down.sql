ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_delivery_status_check;
ALTER TABLE invoices DROP COLUMN IF EXISTS delivery_status;
ALTER TABLE company DROP COLUMN IF EXISTS track_delivery;
