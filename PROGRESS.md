# CostBook API — ความคืบหน้า

> ทำ API **ทีละส่วนตามที่ผู้ใช้สั่งเท่านั้น** ห้ามเริ่มเขียนโค้ดเองล่วงหน้า
> ทุกครั้งที่ทำงานเสร็จ: ติ๊ก `[x]` ในรายการ และเพิ่ม 1 บรรทัดใน "บันทึกการทำงาน" (ล่าสุดอยู่บนสุด)

## สถานะตอนนี้

- **เสร็จแล้ว:** Phase 0 (เอกสาร) และตั้งโครงโฟลเดอร์ของ Phase 1
- **ยังไม่มีโค้ด Go** — รอผู้ใช้สั่งทีละส่วน

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

พื้นฐาน API (รอคำสั่ง):

- [ ] เพิ่ม package `github.com/gin-gonic/gin` (ผู้ใช้อนุมัติแล้ว — เพิ่มตอนเริ่มเขียน router)
- [ ] `cmd/api/main.go` เปิด server
- [ ] `config` อ่าน `PORT`, `ALLOWED_ORIGIN` + test
- [ ] `router` (Gin) + `GET /api/v1/health` + `NoRoute` ตอบ 404 `not_found` + test
- [ ] `middleware/cors.go` (Gin middleware) + test
- [ ] `helpers`: `WriteError`, `ServerError` + test (ตอบสำเร็จใช้ `c.JSON` ของ Gin)
- [ ] `utils`: `Round2`, `VAT`, `TotalWithVAT`, `PriceBeforeVAT`, `CostPerUnit`, `SellPrice` + test

## Phase 2 — ฐานข้อมูล (รอคำสั่ง)

- [x] ติดตั้ง PostgreSQL + สร้าง database `costbook`
- [ ] ติดตั้ง `golang-migrate`
- [ ] เพิ่ม package `pgx/v5`
- [ ] `internal/database/postgres.go` เชื่อมต่อ DB
- [ ] เพิ่ม `DATABASE_URL` ใน config
- [ ] โหลดค่าจากไฟล์ `.env` ตอนพัฒนา (เช่นผ่าน Makefile)
- [x] migration: `users` → `000001_create_users` (ผู้ใช้รัน SQL เองใน DBeaver)
- [ ] migration: `company`, `products`, `product_costs`, `stock_movements`, `transactions`, `invoices`, `invoice_items`, `invoice_numbers`

## Phase 3 — Features (รอคำสั่ง)

แต่ละ feature: model + `Validate()` + test → repository → handler + test (fakeRepository) → ต่อ route

- [ ] company — ข้อมูลบริษัท
- [ ] product — CRUD สินค้า
- [ ] cost — รายการต้นทุน + คำนวณ `cost_price` ใหม่
- [ ] stock — รับเข้า / เบิกออก / ใกล้หมด
- [ ] transaction — รายรับ-รายจ่าย + สรุปรายเดือน
- [ ] invoice — ออกใบกำกับ (transaction: รันเลข + บันทึก + ตัด stock + บันทึกรายรับ) / ยกเลิก
- [ ] dashboard — ตัวเลขสรุป

## Phase 4 — ความปลอดภัยและคุณภาพ (รอคำสั่ง)

- [ ] login: `users` + bcrypt + JWT
- [ ] `middleware/auth.go` ป้องกันทุก route ยกเว้น health และ login
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
- [ ] ใช้ฐานข้อมูลตัวไหน (ค่าเริ่มต้นใน docs: PostgreSQL)
- [ ] deploy API และ DB ที่ไหน (เช่น Railway / Render / Fly.io / VPS + Neon / Supabase สำหรับ DB)
- [ ] ผู้ใช้คนเดียว หรือหลายคน / หลายบริษัท
- [ ] รูปแบบเลขที่ใบกำกับ (ค่าเริ่มต้นใน docs: `INV-2026-0001` รันใหม่ทุกปี)
- [ ] ยกเลิกใบกำกับแล้ว ลบรายรับ หรือบันทึกรายการกลับรายการ (ค่าเริ่มต้นใน docs: ลบรายรับ + คืน stock)
- [ ] จะเก็บรายการต้นทุนละเอียด (`product_costs`) หรือเก็บแค่ `cost_price` ตัวเดียว

---

## บันทึกการทำงาน

| วันที่ | ทำอะไร | หมายเหตุ |
|---|---|---|
| 2026-10-31 | เปิด `pgcrypto` ใน DB `costbook` + เพิ่มผู้ใช้ทดสอบ `admin@costbook.com` (รหัส `1234`, ชื่อ Admin CostBook) — hash เป็น bcrypt `$2a$10$` เช็ครหัสผ่านแล้วถูก | ใช้ทดสอบในเครื่องเท่านั้น ยังไม่มี column role (ทุกคนสิทธิ์เท่ากัน) |
| 2026-10-31 | `sql/users.sql`: คำสั่ง select / insert / update / delete ของตาราง `users` ไว้ใช้ใน DBeaver — insert / เปลี่ยนรหัสผ่านใช้ `crypt()` ของ `pgcrypto` (bcrypt แบบเดียวกับ Go) — ทดสอบทั้งไฟล์ใน transaction แล้ว ROLLBACK ผ่าน | ผู้ใช้รัน migration `users` เองแล้ว / ยังไม่ได้เปิด `pgcrypto` จริง (ผู้ใช้รันเองครั้งแรก) |
| 2026-10-31 | migration ตาราง `users` (id, email, password_hash, first_name, last_name, created_at, updated_at) — ทดสอบรันใน transaction แล้ว ROLLBACK ผ่าน | ใช้ `IF NOT EXISTS` → รันเองใน DBeaver ก่อน แล้วภายหลังใช้ `golang-migrate` ก็ไม่พัง / ยังไม่มี API login |
| 2026-10-31 | ผู้ใช้ติดตั้ง PostgreSQL 17.11 (Homebrew) เปิด service และสร้าง database `costbook` แล้ว | ยังไม่ได้ลง `golang-migrate` |
| 2026-10-31 | เปลี่ยนจาก `net/http` เป็น **Gin**: แก้ `docs/ARCHITECTURE.md`, `CLAUDE.md`, skill `simple-code` / `unit-test-check` และแผน Phase 1 (ตัด `WriteJSON` ใช้ `c.JSON` แทน, เพิ่ม `RegisterRoutes`, `NoRoute`) และ Phase 4 (ตัด `middleware/logger.go` เพราะ `gin.Default()` มี logger แล้ว) | ยังไม่ได้เพิ่ม package / ยังไม่มีโค้ด — ไม่กระทบ Flutter (path และ JSON เหมือนเดิม) |
| 2026-10-31 | ลบโค้ด Go ทั้งหมดออก (main, config, router, health, CORS, helpers, utils และ test) เหลือแค่เอกสาร + โครงโฟลเดอร์เปล่า | ผู้ใช้จะสั่งทำ API ทีละส่วนเอง |
| 2026-10-31 | สร้างเอกสาร + skill และโครงโปรเจกต์ Go: config, router, health, CORS, helpers, utils พร้อม unit test | ภายหลังลบโค้ดออก (ดูบรรทัดบน) |
