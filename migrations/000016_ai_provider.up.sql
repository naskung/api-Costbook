-- AI ที่ร้านเลือกเองในหน้าตั้งค่า: claude = key ของ Claude / openai = AI เจ้าไหนก็ได้ที่ใช้รูปแบบ API ของ OpenAI
-- (OpenAI, DeepSeek, Groq, OpenRouter, Mistral, Typhoon, Ollama ฯลฯ) ต้องมีที่อยู่ API + ชื่อรุ่น / ว่าง = ยังไม่ได้ตั้ง (ใช้ AI จาก .env)
-- ai_key (เข้ารหัส) ใช้ช่องเดิมของ migration 000009 — key เดิมที่ร้านใส่ไว้เป็นของ Claude ทั้งหมด
ALTER TABLE company ADD COLUMN IF NOT EXISTS ai_provider TEXT NOT NULL DEFAULT '';
ALTER TABLE company ADD COLUMN IF NOT EXISTS ai_base_url TEXT NOT NULL DEFAULT '';
ALTER TABLE company ADD COLUMN IF NOT EXISTS ai_model TEXT NOT NULL DEFAULT '';
UPDATE company SET ai_provider = 'claude' WHERE ai_key <> '' AND ai_provider = '';
