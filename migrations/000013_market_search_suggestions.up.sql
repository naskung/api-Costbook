-- แถบคำค้นแนะนำของ Google (HTML) ของผลเช็คราคาตลาดที่ค้นผ่าน Gemini — ต้องแสดงคู่กับผลเสมอ (กติกาของ Google)
-- ผลที่เช็คด้วย Claude ไม่มี = ''
ALTER TABLE market_checks ADD COLUMN IF NOT EXISTS search_suggestions TEXT NOT NULL DEFAULT '';
