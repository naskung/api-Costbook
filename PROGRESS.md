# CostBook API — ความคืบหน้า

> ทำ API **ทีละส่วนตามที่ผู้ใช้สั่งเท่านั้น** ห้ามเริ่มเขียนโค้ดเองล่วงหน้า
> ทุกครั้งที่ทำงานเสร็จ: ติ๊ก `[x]` ในรายการ และเพิ่ม 1 บรรทัดใน "บันทึกการทำงาน" (ล่าสุดอยู่บนสุด)

## สถานะตอนนี้

- **เสร็จแล้ว:** Phase 0, พื้นฐาน API (server, config, router, CORS, helpers), ต่อ DB, login (`POST /auth/login`, `GET /auth/me`)
- **รันได้แล้ว:** `make run` → `http://localhost:8080/api/v1/health`, เอกสาร Swagger: `http://localhost:8080/docs`
- **feature ที่เสร็จ:** auth, company, product, cost, stock, transaction, invoice, dashboard — **ครบทุกเมนูแล้ว** (ทดสอบกับ DB จริง 58 กรณีผ่าน)
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
- [x] `utils`: `Round2`, `Profit`, `VAT`, `CostPerUnit`, `IsDate`, `IsUUID`, `IsTaxID` + test (`TotalWithVAT` / `PriceBeforeVAT` / `SellPrice` API ไม่ได้ใช้ → อยู่ฝั่ง Flutter อย่างเดียว)
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
- [x] cost — ใบต้นทุน `GET` / `PUT /products/{id}/costs` (บันทึกทั้งใบ) + คำนวณ `cost_price` ใหม่ + test
- [x] stock — คงเหลือ / ใกล้หมด / ประวัติ / รับเข้า / เบิกออก (ของไม่พอ 409) + test
- [x] transaction — รายรับ-รายจ่าย + สรุปรายเดือน (รายรับจากใบกำกับแก้ / ลบไม่ได้ 409) + test
- [x] invoice — ออกใบกำกับ (transaction เดียว: รันเลข + บันทึก + ตัด stock + บันทึกรายรับ) / ยกเลิก (คืน stock + ลบรายรับ) + test
- [x] dashboard — ตัวเลขสรุป + สินค้าใกล้หมด + test

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

---

## บันทึกการทำงาน

| วันที่ | ทำอะไร | หมายเหตุ |
|---|---|---|
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
