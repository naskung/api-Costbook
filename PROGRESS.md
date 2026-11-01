# CostBook API — ความคืบหน้า

> ทำ API **ทีละส่วนตามที่ผู้ใช้สั่งเท่านั้น** ห้ามเริ่มเขียนโค้ดเองล่วงหน้า
> ทุกครั้งที่ทำงานเสร็จ: ติ๊ก `[x]` ในรายการ และเพิ่ม 1 บรรทัดใน "บันทึกการทำงาน" (ล่าสุดอยู่บนสุด)

## สถานะตอนนี้

- **เสร็จแล้ว:** Phase 0, พื้นฐาน API (server, config, router, CORS, helpers), ต่อ DB, login (`POST /auth/login`, `GET /auth/me`)
- **รันได้แล้ว:** `make run` → `http://localhost:8080/api/v1/health`, เอกสาร Swagger: `http://localhost:8080/docs`
- **feature ที่เสร็จ:** auth, company, product, field, cost, stock, transaction, invoice, dashboard, ai — **ครบทุกเมนูแล้ว** + **รองรับร้านหลายแบบ** + **ภาพรวมแบบละเอียด + AI เช็คราคาตลาด / แชท** (ทดสอบกับ DB จริง 58 + 43 + 33 กรณีผ่าน)
- **AI มี 2 ตัว:** ร้านใส่ key ของ Claude เองที่หน้าตั้งค่า (เก็บเข้ารหัสใน DB) หรือ `ANTHROPIC_API_KEY` ใน `.env` → ใช้ Claude (แชท + เช็คราคาตลาด) / ไม่มี key Claude → ใช้ **Gemini ฟรี** (`GEMINI_API_KEY` ใน `.env`) แชทได้แต่ค้นเว็บไม่ได้ → เช็คราคาตลาดไม่ได้
- **AI ยังไม่ได้ลองตอบจริง:** ยังไม่มี key — ทดสอบด้วย Claude / Gemini ปลอมใน test (ลองกับ Anthropic จริงแค่ว่า key ปลอมถูกปฏิเสธ)
- **ยังไม่ได้ทำ:** `golang-migrate`, จำกัดขนาด request body, ไฟล์ `sql/` ของตารางใหม่, integration test ของ repository

---

## Phase 0 — เอกสารและกติกา

- [x] โครงสร้างและกติกา `docs/ARCHITECTURE.md` (รวมคำอธิบายว่า backend ต้องมีอะไรบ้าง)
- [x] `CLAUDE.md`
- [x] skill `simple-code` (Go)
- [x] skill `unit-test-check` (Go)
- [x] `PROGRESS.md` (ไฟล์นี้)

## Phase 1 — โครงโปรเจกต์ และพื้นฐาน API

ตั้งโครง (เสร็จแล้ว):

- [x] `go.mod` (module `github.com/naskung/api-Costbook`)
- [x] `.gitignore`
- [x] โฟลเดอร์ตาม `docs/ARCHITECTURE.md` หัวข้อ 4: `cmd/api`, `internal/{config,database,router,middleware,helpers,utils}`, `internal/features/{company,product,stock,transaction,invoice,dashboard}`, `migrations`
  (ในแต่ละโฟลเดอร์มี `.gitkeep` = ไฟล์ว่างให้ git เก็บโฟลเดอร์เปล่าไว้ ลบได้เมื่อมีไฟล์จริง)

พื้นฐาน API:

- [x] เพิ่ม package `github.com/gin-gonic/gin` v1.12.0
- [x] `cmd/api/main.go` เปิด server
- [x] `config` อ่าน `PORT`, `ALLOWED_ORIGIN`, `DATABASE_URL`, `JWT_SECRET` + test
- [x] `router` (Gin) + `GET /api/v1/health` + `NoRoute` ตอบ 404 `not_found` + test
- [x] `middleware/cors.go` (Gin middleware) + test
- [x] `helpers`: `WriteError`, `ServerError` + test (ตอบสำเร็จใช้ `c.JSON` ของ Gin)
- [x] เอกสาร Swagger: `docs/openapi.yaml` + หน้า `GET /docs` (Swagger UI จาก CDN, ไม่เพิ่ม package) + test เช็คว่าทุก route มีในเอกสาร
- [x] `utils`: `Round2`, `Profit`, `VAT`, `CostPerUnit`, `IsDate`, `IsUUID`, `IsTaxID` + test + `PriceBeforeVAT` (ใช้ตอนราคารวม VAT แล้ว) (`TotalWithVAT` / `SellPrice` API ไม่ได้ใช้ → อยู่ฝั่ง Flutter อย่างเดียว)
- [x] `helpers`: `MonthStart` (อ่าน `?month=`), `UUIDParam` (อ่าน `:id` ไม่ใช่ UUID → 404) + test

## Phase 2 — ฐานข้อมูล

- [x] ติดตั้ง PostgreSQL + สร้าง database `costbook`
- [ ] ติดตั้ง `golang-migrate`
- [x] เพิ่ม package `pgx/v5` v5.12.0
- [x] `internal/database/postgres.go` เชื่อมต่อ DB + test
- [x] เพิ่ม `DATABASE_URL` ใน config
- [x] โหลดค่าจากไฟล์ `.env` ตอนพัฒนา ผ่าน `Makefile` (`make run`) + `.env.example`
- [x] migration: `users` → `000001_create_users` (ผู้ใช้รัน SQL เองใน DBeaver)
- [x] migration: `company` → `000002_create_company` (รันใน DB เครื่องนี้แล้ว) + `sql/company.sql`
- [x] migration: `000003_create_products` (products, product_costs), `000004_create_invoices` (invoices, invoice_items, invoice_numbers), `000005_create_stock_movements`, `000006_create_transactions` — รันใน DB `costbook` แล้ว (ทดสอบขาขึ้น + ขาลงใน transaction แล้ว ROLLBACK ผ่าน)
- [ ] `sql/` สำหรับ DBeaver ของตารางใหม่

## Phase 3 — Features

แต่ละ feature: model + `Validate()` + test → repository → handler + test (fakeRepository) → ต่อ route

- [x] company — ข้อมูลร้าน (`GET` / `PUT /company`) + โลโก้ (`GET` / `PUT` / `DELETE /company/logo`) + test
- [x] product — CRUD สินค้า (ลบสินค้าที่อยู่ในใบกำกับไม่ได้ 409) + test
- [x] product — รูปสินค้า 1 รูป (migration `000010_product_image`): `GET` / `PUT` / `DELETE /products/{id}/image` + `image_version` + test
- [x] cost — ใบต้นทุน `GET` / `PUT /products/{id}/costs` (บันทึกทั้งใบ) + คำนวณ `cost_price` ใหม่ + test
- [x] stock — คงเหลือ / ใกล้หมด / ประวัติ / รับเข้า / เบิกออก (ของไม่พอ 409) + test
- [x] transaction — รายรับ-รายจ่าย + สรุปรายเดือน (รายรับจากใบกำกับแก้ / ลบไม่ได้ 409) + test
- [x] invoice — ออกใบกำกับ (transaction เดียว: รันเลข + บันทึก + ตัด stock + บันทึกรายรับ) / ยกเลิก (คืน stock + ลบรายรับ) + test
- [x] dashboard — ตัวเลขสรุป + สินค้าใกล้หมด + test
- [x] รองรับร้านหลายแบบ (migration `000007_flexible_shop`):
  - [x] รูปแบบร้านใน company: `vat_registered`, `prices_include_vat`, `use_cost_page` (ไม่ส่ง = ใช้ค่าเดิม)
  - [x] สินค้า: รหัส / บาร์โค้ด (ห้ามซ้ำ), หมวด, รายละเอียด, `track_stock` (งานบริการไม่นับสต็อก), ช่องข้อมูลเพิ่มเติม (`custom_fields` JSONB)
  - [x] field — ช่องข้อมูลเพิ่มเติมที่ร้านตั้งเอง `/product-fields` (ข้อความ / ตัวเลข / วันที่ / ตัวเลือก) + test
  - [x] cost — บรรทัดต้นทุนแบบ จำนวน × ราคาต่อหน่วย หรือ % ของต้นทุนอื่น + รายการที่ใช้บ่อย `/cost-presets` + test
  - [x] stock — รับเข้าใส่ต้นทุนต่อหน่วย + ผู้ขาย → คิดต้นทุนเฉลี่ยให้ + test
  - [x] invoice — ใบเสร็จรับเงิน (ร้านไม่จด VAT, เลข `RC-`), ส่วนลดทั้งบิล, ราคารวม VAT แล้ว, ไม่ตัด stock ของงานบริการ + test
  - [ ] ใบกำกับภาษีอย่างย่อ (ผู้ใช้บอกว่ายังไม่ต้อง)
- [x] ภาพรวมแบบละเอียด + AI (migration `000008_ai_market`):
  - [x] invoice — เก็บต้นทุน ณ วันที่ขาย (`invoice_items.cost_price`) ไว้คิดกำไรขั้นต้น
  - [x] dashboard — ยอดขายก่อน VAT / กำไรขั้นต้น / margin / เฉลี่ยต่อบิล, เดือนก่อน, ย้อนหลัง 6 เดือน, รายวัน, ขายดี 5 อันดับ, ตามหมวดสินค้า, รายจ่ายตามหมวด, มูลค่าสต็อก, เทียบราคาตลาด + test
  - [x] ai — เช็คราคาตลาดด้วย Claude + ค้นเว็บ (`/products/{id}/market`) + test (Claude ปลอม)
  - [x] ai — แชทผู้ช่วย อ่านข้อมูลร้าน + เสนองานให้ผู้ใช้กดยืนยัน (`/ai/chat`, `/ai/status`) + test (Claude ปลอม)
  - [x] AI ฟรีเป็นค่าเริ่มต้น + ร้านใส่ key ของ Claude เอง (migration `000009_ai_key`):
    - [x] Gemini ฟรี (เรียก REST ด้วย Go standard library ไม่เพิ่ม package) แชท + เสนองานได้เหมือน Claude / ค้นเว็บไม่ได้ → เช็คราคาตลาดตอบ 409 `ai_search_unavailable` + test (Gemini ปลอม + `httptest`)
    - [x] `PUT /ai/key` (ลอง key กับ Anthropic ก่อนบันทึก) / `DELETE /ai/key`, เก็บแบบเข้ารหัส AES-GCM (`utils/secret.go`) ไม่ส่ง key กลับ ส่งแค่ 4 ตัวท้าย + test
    - [x] `GET /ai/status` บอกว่าตอนนี้ใช้ AI ตัวไหน (provider / source / web_search / has_own_key / key_hint / free_available)
  - [ ] ลองกับ Claude / Gemini จริง (รอ key)

## Phase 4 — ความปลอดภัยและคุณภาพ

- [x] login: `users` + bcrypt + JWT (`features/auth`: `POST /auth/login`, `GET /auth/me`, token อายุ 7 วัน) + test
- [x] `middleware/auth.go` ตรวจ token + test — route ใหม่ที่ต้อง login ให้ใส่ `authRequired` ใน router
- [ ] จำกัดขนาด request body
- [ ] (ไม่บังคับ) integration test ของ repository กับ DB จริง

## Phase 5 — Deploy (รอคำสั่ง)

- [ ] เลือกที่ deploy API + DB
- [ ] ตั้งค่า environment variable บน server (`ALLOWED_ORIGIN` = โดเมนเว็บจริง, `GIN_MODE=release`)
- [ ] HTTPS
- [ ] สำรองข้อมูล DB อัตโนมัติ

---

## เรื่องที่ยังต้องตัดสินใจ

- [x] web framework → **Gin** (ตัดสินใจ 2026-10-31)
- [x] Docker → **ยังไม่ใช้** (เครื่องมี Go + PostgreSQL แล้ว) ค่อยดูอีกทีตอน deploy (Phase 5)
- [ ] เปิดหน้า `/docs` บน server จริงด้วยไหม (ตอนนี้เปิดตลอด)
- [ ] ใช้ฐานข้อมูลตัวไหน (ค่าเริ่มต้นใน docs: PostgreSQL)
- [ ] deploy API และ DB ที่ไหน (เช่น Railway / Render / Fly.io / VPS + Neon / Supabase สำหรับ DB)
- [x] ร้านเดียว หรือหลายร้าน → **1 ระบบ = 1 ร้าน** แต่ละร้านติดตั้งแยก (แบบ local) แก้ข้อมูลร้าน / โลโก้ผ่าน API หน้าเว็บไม่ต้องแก้ (ตัดสินใจ 2026-10-31)
- [x] เก็บโลโก้ที่ไหน → ใน DB (`BYTEA`) ไม่เกิน 1 MB, PNG / JPG / WebP — ถ้าวันหลังมีรูปสินค้าเยอะค่อยย้ายไปบริการฝากไฟล์ (R2 / S3)
- [ ] ผู้ใช้หลายคนในร้านเดียว แบ่งสิทธิ์ (admin / พนักงาน) ไหม
- [x] รูปแบบเลขที่ใบกำกับ → `INV-2026-0001` รันใหม่ทุกปี (ตามปีของวันที่ออกใบ) (ตัดสินใจ 2026-10-31)
- [x] ยกเลิกใบกำกับ → ลบรายรับ + คืน stock, ใบกำกับแก้ / ลบไม่ได้, ร้านต้องมีเลขผู้เสียภาษีก่อนออกใบ (ตัดสินใจ 2026-10-31)
- [x] เก็บต้นทุนละเอียด (`product_costs`) → คำนวณ `cost_price` ให้ (ตัดสินใจ 2026-10-31)
- [x] AI → Claude (`claude-opus-5-5`) ผ่าน SDK ทางการ / แชท AI แก้ข้อมูลได้ **แต่ต้องกดยืนยันก่อน** (ลบ / ออกใบ / ยกเลิกใบไม่ได้) / เช็คราคาตลาด **กดทีละสินค้าแล้วเก็บผล** / ยังไม่มี API key ทำโค้ดไว้ก่อน (ตัดสินใจ 2026-10-31)
- [x] ร้านอยากเพิ่มข้อมูลสินค้าเอง → **ช่องข้อมูลเพิ่มเติม** เก็บเป็น JSONB ไม่ ALTER TABLE / ต้นทุน → **จำนวน × ราคาต่อหน่วย + %** + รายการที่ใช้บ่อย / ใบกำกับอย่างย่อ → ยังไม่ทำ (ตัดสินใจ 2026-10-31)

---

## บันทึกการทำงาน

| วันที่ | ทำอะไร | หมายเหตุ |
|---|---|---|
| 2026-11-01 | **แก้การคิดกำไร** (ผู้ใช้แจ้งว่า "กำไร" แสดงเท่ายอดขาย ซึ่งคือรายได้): เดิม กำไร = รายรับ − รายจ่าย ไม่ได้หักต้นทุนสินค้าที่ขาย → ใหม่ `transaction.NewSummary` (รายได้ = ยอดขายก่อน VAT + รายรับที่บันทึกเอง, กำไรขั้นต้น, รายจ่ายที่หักในกำไร = ไม่รวมหมวดซื้อวัตถุดิบ, กำไรสุทธิ) + SQL `transaction.MonthAmountsSQL` ใช้ร่วมกันทั้ง `/transactions/summary` และ `/dashboard` (ตัวเดือน + กราฟ 6 เดือน), `utils.Profit` → `utils.NetProfit`, field ใหม่ `sales` / `other_income` / `revenue` / `cost_of_sales` / `gross_profit` / `material_expense` / `operating_expense` (summary), `revenue` / `other_income` / `material_expense` / `operating_expense` (dashboard), `revenue` / `cost_of_sales` / `operating_expense` (monthly_trend) — `profit` ทุกที่ = กำไรสุทธิ, คำอธิบายเครื่องมือ `get_dashboard` ของ AI, `openapi.yaml` 0.7.0 — gofmt / vet / test ผ่านทุก package — ลองกับ DB ทดสอบ (ขาย 500 + VAT 35, รายรับอื่น 300, ค่าเช่า 500, ซื้อวัตถุดิบ 1,250): รายได้ 800, ต้นทุน 114.84, รายจ่าย 500, กำไรสุทธิ 185.16 ตรงกับคิดมือ และ summary = dashboard = กราฟ ✓ / ข้อมูลจริงเดือน พ.ย.: รายได้ 239.13 − ต้นทุน 227.74 = กำไร 11.39 (เดิมแสดง 239.13) | ผู้ใช้เลือก: ไม่หักซื้อวัตถุดิบซ้ำ / **ความหมายของ `profit` เปลี่ยน** / **ต้อง restart `make run`** / **กระทบ Flutter** ใช้แล้วในหน้ารายรับ-รายจ่าย + ภาพรวม |
| 2026-11-01 | **รูปสินค้า** (ผู้ใช้เลือก 1 รูป / สินค้า): migration `000010_product_image` (`image` BYTEA, `image_content_type`, `image_version` — รันใน DB `costbook` แล้ว, ลอง up ซ้ำ → down → up ใน transaction แล้ว ROLLBACK ผ่าน), product `ImageVersion` + `FindImage` / `SaveImage` / `DeleteImage`, route `GET` (ต้อง login, cache นานเพราะ URL มี `?v=`) / `PUT` (multipart ช่อง `image`) / `DELETE /products/{id}/image`, error ใหม่ `product_image_required` / `_too_large` / `_type_invalid` / `_not_found`, ย้ายตัวอ่านรูปของโลโก้มาไว้กลาง `helpers.ReadImage` + `utils.ImageContentType` (โลโก้กับรูปสินค้าใช้ตัวเดียวกัน, ลบ `company/logic.go`), `openapi.yaml` 0.6.0 (28 path) — gofmt / vet / test ผ่านทุก package — ลองจริงกับ DB ทดสอบ: อัปโหลดผ่านหน้าเว็บ (รูป 2000 × 1600 ถูกย่อเหลือ 1200 × 960, 28 KB) ✓, ไม่มี token 401 ✓, ไฟล์ไม่ใช่รูป 400 ✓, ใหญ่เกิน 400 ✓, ลบรูปแล้ว 404 `product_image_not_found` ✓, preflight 204 ✓ | **ต้อง restart `make run`** (API ที่รันอยู่ยังไม่มีเส้นรูป) / **กระทบ Flutter**: field `image_version` + เส้นรูปใหม่ ใช้แล้วในหน้าสินค้า / ขายด่วน |
| 2026-11-01 | ใส่สินค้าตัวอย่าง 2 ตัวในร้าน (DB เครื่องนี้ผ่าน API ตามที่ผู้ใช้ขอ): `P3D-001` เคสมือถือ iPhone 13 (TPU, ขาย 250, ใบต้นทุนจากรายการที่ใช้บ่อย → ต้นทุน 58.70/ชิ้น) และ `P3D-002` พวงกุญแจ (PLA, ขาย 59, พิมพ์ 6 ชิ้น/รอบ → 18.98/ชิ้น) + ช่องวัสดุ / ประเภทงาน — ต้นทุนที่ API คิดตรงกับที่คิดมือ | ราคาขาย / น้ำหนัก / เวลาพิมพ์เป็น **ตัวอย่าง** / สต็อกยังเป็น 0 / ยังไม่มีรูปสินค้า (ยังไม่มีระบบรูป) |
| 2026-10-31 | ใส่ข้อมูลร้าน 3D print (MAKEHUB) ใน DB เครื่องนี้ผ่าน API: รายการต้นทุนที่ใช้บ่อย 11 รายการ (เส้น PLA / PETG / TPU ต่อกรัม, อุปกรณ์ประกอบ, สี / กาว, บรรจุภัณฑ์, ค่าแรงงานมือต่อนาที, ค่าออกแบบต่อชั่วโมง, ค่าไฟ / ค่าเสื่อมเครื่องต่อชั่วโมง, เผื่อพิมพ์เสีย 10%) + ช่องข้อมูลสินค้า 7 ช่อง (วัสดุ, สี, ประเภทงาน, ขนาด, ค่าที่ใช้พิมพ์, ที่มาของโมเดล, ลิงก์ไฟล์) — เช็คใน DB แล้วครบ | ราคาเป็น **ตัวอย่าง** รอผู้ใช้ส่งราคาจริงมาแก้ / ไม่ได้แก้โค้ด |
| 2026-10-31 | **แชท AI ลอยทุกหน้า + ใส่ key เอง + AI ฟรีเป็นค่าเริ่มต้น** (ผู้ใช้เลือก Gemini ฟรี): migration `000009_ai_key` (`company.ai_key`, รันใน DB `costbook` แล้ว — ลอง up → up ซ้ำ → down → up ใน transaction แล้ว ROLLBACK ผ่าน), `utils/secret.go` (`Encrypt` / `Decrypt` AES-256-GCM, กุญแจ = SHA-256 ของ `JWT_SECRET`) + test, `ai/assistant.go` (`Assistant` interface: Claude / Gemini, `pickProvider` ลำดับ: key ของร้าน → `ANTHROPIC_API_KEY` → `GEMINI_API_KEY` → ปิด, `validClaudeKey`, `keyHint`), `ai/gemini.go` (REST `generateContent` + function calling, ส่ง `thoughtSignature` กลับครบ, รับมือคำตอบที่ถูกบล็อก), `ai/tools.go` เครื่องมือชุดเดียวแปลงได้ทั้ง Claude / Gemini, เครื่องมือ `propose_market_check` ตอบ error เมื่อ AI ค้นเว็บไม่ได้, `Claude.CheckKey` (ลอง key ด้วย `Models.List`), handler ใหม่ `PUT` / `DELETE /ai/key`, error ใหม่ `ai_key_format_invalid` / `ai_quota_exceeded` / `ai_search_unavailable`, config `GEMINI_API_KEY` / `GEMINI_MODEL` (ค่าเริ่มต้น `gemini-3.8-flash`), `.env.example`, `openapi.yaml` 0.5.0 (27 path) — gofmt / vet / test ผ่านทุก package (ai 80.4%, utils 93.8%) — ลองจริงกับ Anthropic: key ปลอมได้ 400 `ai_key_invalid` ไม่ถูกบันทึก, `GET /company` ไม่มี `ai_key` หลุดออกไป | ยังไม่ได้ลองให้ Claude / Gemini ตอบจริง (ไม่มี key) / ถ้าเปลี่ยน `JWT_SECRET` → key ที่ร้านใส่ไว้ถอดไม่ได้ ระบบจะถือว่าไม่มี key (ใส่ใหม่) / Gemini ฟรี: Google อาจเอาข้อความไปพัฒนาบริการ / **กระทบ Flutter**: เส้นใหม่ `/ai/key` + field ใหม่ใน `/ai/status` ใช้แล้วในหน้าตั้งค่า + แชทลอย / **ต้อง restart `make run`** |
| 2026-10-31 | หน้าภาพรวมแบบใหม่ (ผู้ใช้ส่งตัวอย่างมา): `GET /dashboard` เพิ่ม `bills` + `gross_profit` ในแต่ละเดือน และ `previous_daily_sales` (ยอดรายวันของเดือนก่อน) + `previousMonthStart` ใน `dashboard/logic.go` + test, `openapi.yaml` — ทดสอบกับ DB จริง: ชุด AI / ภาพรวม 33 กรณี + ฟิลด์ใหม่ 4 กรณีผ่าน (ม.ค. → ธ.ค. ปีก่อน 31 วัน ✓) | **กระทบ Flutter**: field ใหม่ ใช้แล้วในหน้าภาพรวมใหม่ / ต้อง restart `make run` |
| 2026-10-31 | **ภาพรวมแบบละเอียด + AI**: เพิ่ม package `anthropic-sdk-go/v2` v2.4.0 (ผู้ใช้อนุญาต), migration `000008_ai_market` (รันใน DB `costbook` แล้ว — ทดสอบ up → up ซ้ำ → down → up ใน transaction แล้ว ROLLBACK ผ่าน), invoice เก็บ `cost_price` ณ วันที่ขาย, dashboard ส่งข้อมูลกราฟ / กำไรขั้นต้น / ขายดี / ตามหมวด / มูลค่าสต็อก / เทียบตลาด (`dashboard/logic.go` + test), feature ใหม่ `ai`: เช็คราคาตลาด (web search + เครื่องมือ `save_market_report` แบบ strict, รับมือ `pause_turn`, เตือนเมื่อลืมส่งผล) + แชท (เครื่องมืออ่านข้อมูล 6 ตัว, เสนองาน 6 ตัว ตรวจด้วย `Validate()` เดิม), config `ANTHROPIC_API_KEY` (ไม่ใส่ = ปิด AI), `openapi.yaml` 0.4.0 (26 path / 41 operation) — gofmt / vet / test ผ่านทุก package — e2e กับ DB ชั่วคราว: ชุดใหม่ 33 + ชุดเดิม 58 + 43 กรณีผ่าน | ยังไม่ได้เรียก Claude จริง (ไม่มี API key) — test ใช้ Claude ปลอม / e2e ครั้งแรกเจอ 1 กรณีที่ **ผมคิดค่าที่คาดไว้ผิดเอง** (% ตามหมวด 79.34 ไม่ใช่ 79.35) โค้ดถูก แก้ค่าในสคริปต์ทดสอบ / **กระทบ Flutter**: field ใหม่ใน `GET /dashboard` + เส้น AI ใหม่ บันทึกใน PROGRESS ฝั่ง Flutter แล้ว / **ต้อง restart `make run`** |
| 2026-10-31 | **รองรับร้านหลายแบบ**: migration `000007_flexible_shop` (รันใน DB `costbook` แล้ว — ตารางยังว่าง, ทดสอบ up → up ซ้ำ → down → up ใน transaction แล้ว ROLLBACK ผ่าน) + company (รูปแบบร้าน 3 ค่า), product (code / category / description / track_stock / custom_fields), feature ใหม่ `field` (`/product-fields`), cost (kind fixed / percent, quantity × unit_price, `/cost-presets`), stock (unit_cost + supplier → ต้นทุนเฉลี่ย, `FOR UPDATE`), invoice (doc_type receipt / tax_invoice, เลข `RC-` แยก, discount, prices_include_vat, stock_cut), dashboard (ใกล้หมดนับเฉพาะ track_stock), `utils.PriceBeforeVAT` + `openapi.yaml` 0.3.0 (23 path / 37 operation) — gofmt / vet / test ผ่านทุก package — e2e กับ DB ชั่วคราว `costbook_e2e`: ชุดเดิม 58 + ชุดใหม่ 43 กรณีผ่าน | e2e ชุดเดิมพัง 12 กรณีตอนแรก: `PUT /company` ที่ไม่ส่งค่าใหม่ทำให้ `vat_registered` กลายเป็น false → แก้เป็น `*bool` + `COALESCE` (ไม่ส่ง = ค่าเดิม) ทำแบบเดียวกันกับ track_stock / custom_fields / **กระทบ Flutter**: field ใหม่หลายตัว + `items[].amount` ในใบต้นทุนเลิกรับ (ใช้ quantity × unit_price) บันทึกใน PROGRESS ฝั่ง Flutter แล้ว / **ต้อง restart `make run`** |
| 2026-10-31 | **API ทุกเมนู**: product, cost, stock, transaction, invoice, dashboard (model + Validate + logic + repository + handler + test) + migration 000003–000006 (รันใน DB `costbook` แล้ว) + `utils` (Round2, Profit, VAT, CostPerUnit, IsDate, IsUUID, IsTaxID — ย้าย isTaxID จาก company มาไว้ที่นี่) + `helpers` (MonthStart, UUIDParam) + `openapi.yaml` ครบ 29 endpoint (test เช็ค route แปลง `:id` → `{id}` แล้ว) — gofmt / vet / test ผ่านทุก package — ทดสอบจริงกับ DB ชั่วคราว `costbook_e2e` 58 กรณีผ่าน (VAT ทั้งใบ, รันเลข + ขึ้นปีใหม่, ร้านไม่มีเลขภาษี / ของไม่พอแล้ว rollback ไม่ตัด stock, ยกเลิกใบคืน stock + ลบรายรับ, ลบสินค้าในใบไม่ได้, รายรับจากใบลบไม่ได้) | e2e เจอบั๊กลำดับบรรทัดในใบกำกับสลับ (created_at เท่ากันใน transaction) → เพิ่ม `line_no` ใน invoice_items / product_costs (ตารางยังว่าง สร้างใหม่ใน DB แล้ว) / **กระทบ Flutter**: API ใหม่ทั้งหมด บันทึกใน PROGRESS ฝั่ง Flutter แล้ว |
| 2026-10-31 | ใส่ข้อมูลร้านจริงใน DB เครื่องนี้ผ่าน API: MAKEHUB, 33/3 ม.3 ต.ท่าพริก อ.เมือง จ.ตราด 23000, 0909966818 + โลโก้ (PNG 1254×1254, 380 KB) — โหลดกลับมาได้ไฟล์ตรงกัน | ยังไม่มีเลขผู้เสียภาษี / สาขา (ต้องมีก่อนออกใบกำกับภาษี) |
| 2026-10-31 | **ข้อมูลร้าน + โลโก้**: migration `000002_create_company` (แถวเดียว id = 1, logo `BYTEA`) — รันใน DB แล้ว, `sql/company.sql`, `features/company` (model + Validate, logic `logoContentType`, repository, handler), ต่อ route, เพิ่มใน `openapi.yaml` — test ผ่านทุก package (company 80.8%, ส่วนที่ขาดคือ repository) — ทดสอบจริงกับ DB 13 กรณีผ่าน: ดูข้อมูล / แก้ข้อมูล / ไม่ login ได้ 401 / เลขภาษีผิด 400 / อัปโหลด PNG แล้วโหลดกลับได้ไฟล์ตรงกัน / 2 MB ได้ `logo_too_large` / ไฟล์ข้อความได้ `logo_type_invalid` / ลบโลโก้แล้วได้ 404 / preflight 204 — ล้างข้อมูลทดสอบออกแล้ว (ร้านยังว่าง) | **กระทบ Flutter**: API ใหม่ บันทึกใน `PROGRESS.md` ฝั่ง Flutter แล้ว / ดู + โลโก้ไม่ต้อง login (ให้หน้า login แสดงได้) |
| 2026-10-31 | เอกสาร Swagger: `docs/openapi.yaml` (health, auth/login, auth/me + Authorize ด้วย Bearer token), `docs/swagger.html`, `docs/docs.go` (go:embed → `GET /docs`, `GET /docs/openapi.yaml`) + test (`docs` 100%, router เพิ่ม `TestRouter_Docs`, `TestRouter_AllRoutesInOpenAPI`) — ตรวจ YAML + `$ref` ทั้งไฟล์ด้วย pyyaml ผ่าน (เจอและแก้ description ที่มี `: ` ไม่ได้ครอบ `"`), รันจริง `/docs` ได้ 200 HTML | ไม่เพิ่ม package — Swagger UI โหลดจาก unpkg (ต้องต่อเน็ตตอนเปิดดู) / test ใน Go เช็คแค่ว่ามี path ครบ ไม่ได้เช็คว่า YAML ถูกรูปแบบ |
| 2026-10-31 | **API login ใช้งานได้**: เพิ่ม package gin / pgx / golang-jwt / x/crypto, เขียน `main.go`, `config`, `database`, `helpers`, `middleware` (cors, auth), `utils/token.go`, `router`, `features/auth` (model, repository, handler), `Makefile`, `.env.example` + `.env` (JWT_SECRET สุ่มด้วย openssl) — gofmt / vet ผ่าน, test ผ่านทุก package (config / helpers / middleware / router / utils 100%, database 87.5%, auth 77.4% — ส่วนที่ขาดคือ repository ต้องใช้ DB จริง) — ลองรันจริงกับ DB: login `admin@costbook.com` / `1234` ได้ token, รหัสผิด / ไม่มีอีเมล ได้ 401 `invalid_credentials`, `/auth/me` มี token ได้ 200 ไม่มีได้ 401, OPTIONS ได้ 204 | **กระทบ Flutter**: API ใหม่ บันทึกใน `PROGRESS.md` ฝั่ง Flutter แล้ว |
| 2026-10-31 | เปิด `pgcrypto` ใน DB `costbook` + เพิ่มผู้ใช้ทดสอบ `admin@costbook.com` (รหัส `1234`, ชื่อ Admin CostBook) — hash เป็น bcrypt `$2a$10$` เช็ครหัสผ่านแล้วถูก | ใช้ทดสอบในเครื่องเท่านั้น ยังไม่มี column role (ทุกคนสิทธิ์เท่ากัน) |
| 2026-10-31 | `sql/users.sql`: คำสั่ง select / insert / update / delete ของตาราง `users` ไว้ใช้ใน DBeaver — insert / เปลี่ยนรหัสผ่านใช้ `crypt()` ของ `pgcrypto` (bcrypt แบบเดียวกับ Go) — ทดสอบทั้งไฟล์ใน transaction แล้ว ROLLBACK ผ่าน | ผู้ใช้รัน migration `users` เองแล้ว / ยังไม่ได้เปิด `pgcrypto` จริง (ผู้ใช้รันเองครั้งแรก) |
| 2026-10-31 | migration ตาราง `users` (id, email, password_hash, first_name, last_name, created_at, updated_at) — ทดสอบรันใน transaction แล้ว ROLLBACK ผ่าน | ใช้ `IF NOT EXISTS` → รันเองใน DBeaver ก่อน แล้วภายหลังใช้ `golang-migrate` ก็ไม่พัง / ยังไม่มี API login |
| 2026-10-31 | ผู้ใช้ติดตั้ง PostgreSQL 17.11 (Homebrew) เปิด service และสร้าง database `costbook` แล้ว | ยังไม่ได้ลง `golang-migrate` |
| 2026-10-31 | เปลี่ยนจาก `net/http` เป็น **Gin**: แก้ `docs/ARCHITECTURE.md`, `CLAUDE.md`, skill `simple-code` / `unit-test-check` และแผน Phase 1 (ตัด `WriteJSON` ใช้ `c.JSON` แทน, เพิ่ม `RegisterRoutes`, `NoRoute`) และ Phase 4 (ตัด `middleware/logger.go` เพราะ `gin.Default()` มี logger แล้ว) | ยังไม่ได้เพิ่ม package / ยังไม่มีโค้ด — ไม่กระทบ Flutter (path และ JSON เหมือนเดิม) |
| 2026-10-31 | ลบโค้ด Go ทั้งหมดออก (main, config, router, health, CORS, helpers, utils และ test) เหลือแค่เอกสาร + โครงโฟลเดอร์เปล่า | ผู้ใช้จะสั่งทำ API ทีละส่วนเอง |
| 2026-10-31 | สร้างเอกสาร + skill และโครงโปรเจกต์ Go: config, router, health, CORS, helpers, utils พร้อม unit test | ภายหลังลบโค้ดออก (ดูบรรทัดบน) |
