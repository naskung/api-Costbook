-- ย้อนกลับ: ลบผลเช็คราคาตลาด และต้นทุน ณ วันที่ขาย
DROP TABLE IF EXISTS market_checks;
ALTER TABLE invoice_items DROP COLUMN IF EXISTS cost_price;
