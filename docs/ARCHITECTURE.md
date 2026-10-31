# CostBook API — โครงสร้างโปรเจกต์ (Go)

> ไฟล์นี้คือ **กติกากลาง** ของ API ทุกคน (รวมถึง Claude) ต้องทำตามนี้
> ถ้าจะเปลี่ยนกติกา ให้แก้ไฟล์นี้ก่อน แล้วค่อยแก้โค้ด
> หลักสำคัญที่สุด: **เขียนให้ง่าย (simple)** ดู `.claude/skills/simple-code/SKILL.md`

---

## 1. Backend คืออะไร ต้องมีอะไรบ้าง (สำหรับคนที่ยังไม่เคยทำ)

Backend คือโปรแกรมที่รันอยู่บน server คอยรับคำขอ (request) จากแอป Flutter ไปอ่าน / เขียนฐานข้อมูล แล้วตอบกลับ (response) เป็น JSON

```
Flutter Web ──HTTP request──▶ Go API ──SQL──▶ PostgreSQL
            ◀──JSON response──        ◀──ข้อมูล──
```

| สิ่งที่ต้องมี | คืออะไร (ภาษาง่าย) | ในโปรเจกต์นี้ใช้ | Phase |
|---|---|---|---|
| Web framework | ชุดเครื่องมือสำเร็จรูปสำหรับทำ API: เปิด server, router, อ่าน / ตอบ JSON, middleware | **Gin** (`github.com/gin-gonic/gin`) | 1 |
| HTTP server | โปรแกรมที่เปิดรอรับ request ที่ port หนึ่ง | `r.Run(":8080")` ของ Gin (ข้างในใช้ `net/http`) | 1 |
| Router | จับคู่ "method + URL" กับฟังก์ชันที่จะทำงาน | `r.GET(...)`, `r.POST(...)` ของ Gin | 1 |
| JSON | รูปแบบข้อมูลที่ส่งไปกลับกับ Flutter | `c.ShouldBindJSON` (อ่าน) / `c.JSON` (ตอบ) ของ Gin | 1 |
| CORS | อนุญาตให้เว็บที่อยู่คนละ port / โดเมนเรียก API ได้ ถ้าไม่มี browser จะบล็อก | `middleware/cors.go` | 1 |
| Error format | ตอบ error แบบเดียวกันทุกที่ Flutter อ่านง่าย | `{"error": "code"}` | 1 |
| Config | ค่าที่ต่างกันแต่ละเครื่อง (port, รหัส DB) อ่านจาก environment variable ไม่เขียนลงโค้ด | `config/config.go` | 1 |
| Unit test | test อัตโนมัติ เช็คว่าโค้ดยังถูกอยู่ | `testing` + `httptest` (มากับ Go) + `gin.TestMode` | 1 และทุก Phase |
| Database | ที่เก็บข้อมูลถาวร | PostgreSQL + `pgx` | 2 |
| Migration | ไฟล์ SQL สร้าง / แก้ตาราง เรียงตามลำดับ ทุกเครื่องได้ตารางเหมือนกัน | `golang-migrate` | 2 |
| Handler | ฟังก์ชันที่รับ request แล้วตอบกลับ (1 endpoint = 1 ฟังก์ชัน) | `features/*/handler.go` | 3 |
| Validation | เช็คข้อมูลที่ส่งมาก่อนบันทึก เช่น ชื่อห้ามว่าง ราคาห้ามติดลบ | `Validate()` ใน model | 3 |
| DB Transaction | หลายคำสั่ง SQL ต้องสำเร็จพร้อมกัน หรือยกเลิกทั้งหมด (เช่น ออกใบกำกับ + ตัด stock) | `pgx` Begin / Commit | 3 |
| Auth | login และเช็คว่าใครเรียก API | JWT (`golang-jwt`) + bcrypt — token อายุ 7 วัน ส่งมาใน header `Authorization: Bearer <token>` | 4 (เสร็จแล้ว) |
| Logging | พิมพ์ว่ามี request อะไรเข้ามา ไว้หาจุดผิด | log ของ request: มากับ `gin.Default()` / log ของ error: `log/slog` (มากับ Go) | 1 |
| Deploy | เอา API ขึ้น server จริง + HTTPS + สำรองข้อมูล | ยังไม่เลือก | 5 |

---

## 2. เครื่องมือที่ต้องติดตั้ง

| เครื่องมือ | สถานะ | วิธีติดตั้ง |
|---|---|---|
| Go | มีแล้ว (1.26) | — |
| PostgreSQL | มีแล้ว (17.11) | `brew install postgresql@17` → เพิ่ม `export PATH="/opt/homebrew/opt/postgresql@17/bin:$PATH"` ใน `~/.zshrc` → `brew services start postgresql@17` → `createdb costbook` |
| golang-migrate | ยังไม่มี (Phase 2) | `brew install golang-migrate` |
| ตัวยิง API ทดสอบ | มี `curl` แล้ว | หรือใช้ Postman / Bruno ถ้าอยากได้หน้าจอ |
| ตัวดูข้อมูลใน DB | มีแล้ว (DBeaver) | — |
| VS Code extension | แนะนำ | "Go" (ของทีม Go) — จัดรูปแบบโค้ดและรัน test ได้ในตัว |

---

## 3. Package ที่ใช้

| Package | ใช้ทำอะไร | Phase |
|---|---|---|
| `github.com/gin-gonic/gin` | web framework: server, router, อ่าน / ตอบ JSON, middleware | 1 |
| Go standard library | test, log error, อ่าน environment variable และอื่น ๆ | 1 |
| `github.com/jackc/pgx/v5` | ต่อ PostgreSQL | 2 |
| `golang-migrate` (CLI ไม่ใช่ package) | รันไฟล์ migration | 2 |
| `github.com/golang-jwt/jwt/v5` | token สำหรับ login | 4 |
| `golang.org/x/crypto/bcrypt` | เข้ารหัสรหัสผ่าน | 4 |

**ใช้ Gin** (ผู้ใช้เลือก 2026-10-31) เพราะคนใช้เยอะที่สุดใน Go หาตัวอย่าง / บทความ / วิดีโอสอนได้ง่าย
ใช้แค่ router, อ่าน / ตอบ JSON, middleware — **ไม่ใช้** `binding` tag / validator ของ Gin (ใช้ `Validate()` ของเราเอง เพื่อคืน error code ตามที่ Flutter ใช้)

**ไม่ใช้:** Echo / Fiber / Chi, GORM (ORM), sqlc, wire, testify
เหตุผล: ใช้ framework ตัวเดียวพอ และอ่าน SQL ตรง ๆ เข้าใจง่ายกว่า ORM

---

## 4. โครงสร้างโฟลเดอร์

```
api-Costbook/
├── cmd/api/main.go           จุดเริ่มโปรแกรม: อ่าน config → สร้าง router → เปิด server
├── internal/                 (Go ไม่ให้โปรเจกต์อื่น import โค้ดในนี้)
│   ├── config/               อ่านค่าจาก environment variable
│   ├── database/             เชื่อมต่อ PostgreSQL (Phase 2)
│   ├── router/               สร้าง Gin router + รวม route ทั้งหมดไว้ที่เดียว
│   ├── middleware/           ทำงานก่อนทุก request: cors, auth (logger มากับ gin.Default() แล้ว)
│   ├── helpers/              ช่วยตอบ error: WriteError, ServerError (ตอบสำเร็จใช้ c.JSON ของ Gin)
│   ├── utils/                ฟังก์ชันล้วน ๆ ไม่แตะ HTTP / DB: money, tax, cost, token (สร้าง / ตรวจ JWT)
│   └── features/
│       ├── auth/             login + ข้อมูลผู้ใช้ที่ login อยู่ (ตาราง users)
│       ├── company/          ข้อมูลร้าน + โลโก้ (ตาราง company มีแถวเดียว)
│       ├── product/
│       │   ├── model.go           struct + Validate()
│       │   ├── model_test.go
│       │   ├── repository.go      คุยกับ DB (interface + Postgres)
│       │   ├── handler.go         รับ request / ตอบ JSON + RegisterRoutes (ต่อ route ของ feature)
│       │   ├── handler_test.go    test ด้วย fakeRepository (ไม่ต้องมี DB)
│       │   ├── logic.go           (ถ้ามี) คำนวณเฉพาะ feature นี้
│       │   └── logic_test.go
│       ├── stock/
│       ├── transaction/
│       ├── invoice/
│       └── dashboard/
├── migrations/               ไฟล์ SQL สร้าง / แก้ตาราง เช่น 000001_create_users.up.sql / .down.sql
├── sql/                      คำสั่ง SQL ไว้ copy ไปรันเองใน DBeaver (select / insert / update / delete) 1 ไฟล์ต่อ 1 ตาราง — API ไม่ได้ใช้
├── docs/
│   ├── ARCHITECTURE.md       ไฟล์นี้
│   ├── openapi.yaml          เอกสาร API (Swagger) — เพิ่ม / แก้ endpoint ต้องแก้ไฟล์นี้ด้วย
│   ├── swagger.html          หน้าเว็บ Swagger UI (โหลดจาก CDN)
│   └── docs.go               ฝัง 2 ไฟล์ข้างบนเข้าโปรแกรม (go:embed) แล้วเปิดที่ GET /docs
├── PROGRESS.md               ทำถึงไหนแล้ว
├── Makefile                  คำสั่งลัด: make run / make test / make check (โหลดค่าจาก .env ให้)
├── .env.example              ตัวอย่างค่า config → copy เป็น .env (ห้าม commit .env)
└── go.mod
```

ทุก feature ใช้ไฟล์ชื่อเดียวกัน: `model.go / repository.go / handler.go / logic.go` (ไม่มีอันไหนก็ไม่ต้องสร้าง)

### ไฟล์นี้ควรอยู่ไหน?

| ถ้าเป็นโค้ดที่... | วางที่ |
|---|---|
| คำนวณ ไม่แตะ HTTP / DB และใช้หลาย feature | `internal/utils/` |
| คำนวณ ไม่แตะ HTTP / DB และใช้ feature เดียว | `features/<ชื่อ>/logic.go` |
| ช่วยเขียน response / อ่าน request | `internal/helpers/` |
| ต้องทำกับทุก request | `internal/middleware/` |
| คำสั่ง SQL | `features/<ชื่อ>/repository.go` เท่านั้น |
| อ่าน request + ตอบ response | `features/<ชื่อ>/handler.go` |

---

## 5. การไหลของข้อมูล

```
Request → Gin (logger, recovery) → middleware (cors, auth) → handler ──▶ repository ──▶ PostgreSQL
                                                                │
                                                                └─▶ Validate() / logic / utils (คำนวณ)
```

| ชั้น | ทำ | ห้ามทำ |
|---|---|---|
| handler | รับ `*gin.Context`, อ่าน JSON, เรียก `Validate()`, เรียก repository, ตอบ JSON + status | เขียน SQL |
| model | struct + `Validate()` | เรียก DB |
| logic / utils | คำนวณ | แตะ HTTP, Gin หรือ DB |
| repository | SQL และ transaction รับ `context.Context` (handler ส่ง `c.Request.Context()` มา) | รู้จัก `gin.Context` หรือ `http.Request` |

ไม่มีชั้น service แยก — ถ้า feature ไหนต้องทำหลายขั้นใน DB (เช่น ออกใบกำกับ) ให้ทำใน repository ภายใน transaction เดียว

---

## 6. กติกาของ API

- ทุก path ขึ้นต้นด้วย `/api/v1`
- ชื่อ field ใน JSON เป็น `snake_case` และ **ต้องตรงกับ model ฝั่ง Flutter** เช่น `cost_price`, `min_stock`
- `id` เป็น UUID (ส่งเป็น string)
- เงิน: DB เก็บเป็น `NUMERIC(12,2)` ส่วนใน Go ใช้ `float64` และปัดด้วย `utils.Round2` ทุกครั้งที่คำนวณเสร็จ
- วันที่ `"2026-10-31"`, วันเวลา `"2026-10-31T10:00:00Z"`
- รายการว่างต้องตอบ `[]` ไม่ใช่ `null` → สร้าง slice ด้วย `products := []Product{}`
- สำเร็จ: ตอบข้อมูลตรง ๆ (object หรือ array) ด้วย `c.JSON(status, data)` ไม่ห่อด้วย `{"data": ...}`
- error: `{"error": "product_name_required"}` — เป็น **code ภาษาอังกฤษ** ฝั่ง Flutter แปลเป็นข้อความด้วย `'error.<code>'.tr`
  - ตอบ error ผ่าน `helpers.WriteError(c, status, code)` เสมอ (ข้างในใช้ `c.AbortWithStatusJSON` → ใช้ได้ทั้งใน handler และ middleware)

| Method | ใช้ทำ | status เมื่อสำเร็จ |
|---|---|---|
| GET | อ่าน | 200 |
| POST | สร้าง | 201 |
| PUT | แก้ไข | 200 |
| DELETE | ลบ | 204 (ไม่มี body) → `c.Status(http.StatusNoContent)` |

| status error | เมื่อไหร่ | ตัวอย่าง code |
|---|---|---|
| 400 | ข้อมูลที่ส่งมาผิด | `invalid_json`, `product_name_required`, `email_required`, `password_required`, `company_name_required`, `company_tax_id_invalid`, `logo_required`, `logo_too_large`, `logo_type_invalid` |
| 401 | ยังไม่ login / token ผิดหรือหมดอายุ / login ไม่ผ่าน | `unauthorized`, `invalid_credentials` |
| 404 | ไม่เจอข้อมูล / ไม่มี path นี้ | `product_not_found`, `logo_not_found`, `not_found` (จาก `r.NoRoute` ใน router) |
| 409 | ขัดกับข้อมูลที่มีอยู่ | `stock_not_enough`, `product_in_use` |
| 500 | server พัง | `server_error` (ใช้ `helpers.ServerError` → error จริงลง log ไม่ส่งให้ client) |

---

## 7. Endpoints

**เอกสาร Swagger:** `make run` แล้วเปิด http://localhost:8080/docs — ดูทุก endpoint และกด **Try it out** ลองเรียกได้
เพิ่ม / แก้ endpoint → แก้ `docs/openapi.yaml` ด้วยทุกครั้ง (`TestRouter_AllRoutesInOpenAPI` เช็คว่าทุก route ใน `/api/v1` มีในไฟล์นี้ ถ้าลืม test ไม่ผ่าน)
ในไฟล์ YAML ถ้าข้อความมี `: ` (โคลอน + เว้นวรรค) ต้องครอบด้วย `"..."` ไม่งั้นหน้า Swagger โหลดไม่ขึ้น

| Feature | Method + Path | ทำอะไร |
|---|---|---|
| docs | `GET /docs` | หน้าเอกสาร Swagger (ไม่อยู่ใต้ `/api/v1`) |
| health | `GET /api/v1/health` | เช็คว่า API ทำงาน |
| company ✅ | `GET /api/v1/company` | ข้อมูลร้าน (ผู้ขาย) — ไม่ต้อง login |
| | `PUT /api/v1/company` | แก้ข้อมูลร้าน |
| | `GET /api/v1/company/logo` | โลโก้ร้าน (ตอบเป็นรูป) — ไม่ต้อง login |
| | `PUT /api/v1/company/logo` | อัปโหลดโลโก้ (multipart ช่อง `logo`, PNG / JPG / WebP ไม่เกิน 1 MB) |
| | `DELETE /api/v1/company/logo` | ลบโลโก้ |
| product | `GET /api/v1/products` | รายการสินค้า |
| | `GET /api/v1/products/{id}` | สินค้า 1 ตัว |
| | `POST /api/v1/products` | เพิ่มสินค้า |
| | `PUT /api/v1/products/{id}` | แก้สินค้า |
| | `DELETE /api/v1/products/{id}` | ลบสินค้า |
| cost | `GET /api/v1/products/{id}/costs` | รายการต้นทุน (วัตถุดิบ / ค่าแรง / ค่าใช้จ่ายอื่น) |
| | `POST /api/v1/products/{id}/costs` | เพิ่มรายการต้นทุน → คำนวณ `cost_price` ใหม่ |
| | `DELETE /api/v1/products/{id}/costs/{costId}` | ลบรายการต้นทุน → คำนวณ `cost_price` ใหม่ |
| stock | `GET /api/v1/stock?low=true` | จำนวนคงเหลือ (`low=true` = เฉพาะใกล้หมด) |
| | `GET /api/v1/stock/movements?product_id=` | ประวัติรับเข้า / เบิกออก |
| | `POST /api/v1/stock/in` | รับสินค้าเข้า |
| | `POST /api/v1/stock/out` | เบิกสินค้าออก |
| transaction | `GET /api/v1/transactions?month=2026-10` | รายรับ-รายจ่ายของเดือน |
| | `GET /api/v1/transactions/summary?month=2026-10` | สรุปรายรับ รายจ่าย กำไร |
| | `POST /api/v1/transactions` | เพิ่มรายการ |
| | `PUT /api/v1/transactions/{id}` | แก้รายการ |
| | `DELETE /api/v1/transactions/{id}` | ลบรายการ |
| invoice | `GET /api/v1/invoices` | รายการใบกำกับภาษี |
| | `GET /api/v1/invoices/{id}` | ใบกำกับ 1 ใบ (Flutter เอาไปสร้าง PDF) |
| | `POST /api/v1/invoices` | ออกใบกำกับ |
| | `POST /api/v1/invoices/{id}/cancel` | ยกเลิกใบกำกับ |
| dashboard | `GET /api/v1/dashboard?month=2026-10` | ตัวเลขสรุปหน้าแรก |
| auth ✅ | `POST /api/v1/auth/login` | login ได้ token |
| | `GET /api/v1/auth/me` | ข้อมูลผู้ใช้ที่ login อยู่ (ต้องส่ง token) |

ในตารางเขียน path แบบ `{id}` แต่ในโค้ด Gin เขียนเป็น `:id` เช่น `api.GET("/products/:id", ...)` แล้วอ่านค่าด้วย `c.Param("id")`

### รายละเอียด auth (ทำแล้ว)

`POST /api/v1/auth/login`

```json
// request
{ "email": "admin@costbook.com", "password": "1234" }

// 200
{
  "token": "eyJhbGciOi...",
  "user": { "id": "a00d265d-...", "email": "admin@costbook.com", "first_name": "Admin", "last_name": "CostBook" }
}
```

- อีเมลไม่สนตัวพิมพ์เล็ก / ใหญ่ และตัดช่องว่างหน้า-หลังให้ (เก็บใน DB เป็นตัวเล็กเสมอ)
- error: 400 `invalid_json` / `email_required` / `password_required`, 401 `invalid_credentials` (อีเมลไม่มี หรือรหัสผิด ตอบเหมือนกัน)

`GET /api/v1/auth/me` — header `Authorization: Bearer <token>`

- 200 `{ "id", "email", "first_name", "last_name" }`
- 401 `unauthorized` (ไม่มี token / token ผิด / หมดอายุ / ผู้ใช้ถูกลบ) → Flutter พาไปหน้า login

route ใหม่ที่ต้อง login ก่อน → ใส่ `authRequired` (`middleware.Auth`) ใน `router.go` แล้ว handler อ่าน id ผู้ใช้ด้วย `c.GetString(middleware.UserIDKey)`

### รายละเอียด company (ทำแล้ว)

**1 ระบบ = 1 ร้าน** — ร้านใหม่ติดตั้งระบบแยก (แบบ local) แล้วแก้ชื่อ / ที่อยู่ / โลโก้ผ่าน API หน้าเว็บ Flutter ใช้ตัวเดียวกันทุกร้าน ไม่ต้องแก้

- `GET /api/v1/company` → `{ "name", "address", "tax_id", "branch", "phone", "has_logo" }` (ยังไม่ตั้งค่า = ค่าว่างทุกช่อง)
- `PUT /api/v1/company` ส่งครบทุกช่อง (ยกเว้น `has_logo`) — `name` ห้ามว่าง, `tax_id` ว่างได้หรือเป็นตัวเลข 13 หลัก
- `GET /api/v1/company/logo` ตอบเป็นรูปตรง ๆ → `Image.network(...)` ได้เลย / 404 `logo_not_found` → แสดงโลโก้ CostBook แทน
- `PUT /api/v1/company/logo` → ได้ข้อมูลร้าน (`has_logo: true`) — เช็คชนิดไฟล์จากข้างในไฟล์ (`http.DetectContentType`) ไม่เชื่อนามสกุล, ไม่รับ SVG / GIF
- ดูข้อมูลร้าน / โลโก้ไม่ต้อง login เพื่อให้หน้า login แสดงชื่อ + โลโก้ร้านได้
- โลโก้ตอบ header `Cache-Control: no-cache` — Flutter ที่เพิ่งอัปโหลดโลโก้ใหม่ต้องโหลดรูปใหม่เอง (ล้าง cache ของ `Image.network`)

---

## 8. ตารางในฐานข้อมูล

| ตาราง | เก็บอะไร | column หลัก |
|---|---|---|
| `company` | ข้อมูลร้าน (มีแถวเดียว `id = 1`, migration `000002_create_company` สร้างแถวว่างไว้ให้) | name, address, tax_id, branch, phone, logo (`BYTEA` ว่างได้), logo_content_type |
| `products` | สินค้า | id, name, unit, cost_price, sell_price, stock, min_stock, batch_quantity |
| `product_costs` | รายการต้นทุนของสินค้า | id, product_id, type (material / labor / overhead), name, amount |
| `stock_movements` | ประวัติรับเข้า / เบิกออก | id, product_id, type (in / out), quantity, note, invoice_id, created_at |
| `transactions` | รายรับ-รายจ่าย | id, type (income / expense), category, amount, date, note, invoice_id |
| `invoices` | ใบกำกับภาษี | id, invoice_no, issue_date, status, ข้อมูลผู้ขาย + ผู้ซื้อ, subtotal, vat, total |
| `invoice_items` | รายการในใบกำกับ | id, invoice_id, product_id, name, quantity, unit_price, amount |
| `invoice_numbers` | ตัวรันเลขที่ใบกำกับ | year, last_number |
| `users` | ผู้ใช้ที่ login ได้ (migration `000001_create_users`) | id, email (ห้ามซ้ำ), password_hash (bcrypt), first_name, last_name |

- ทุกตารางมี `created_at` และ `updated_at`
- `cost_price` = `utils.CostPerUnit(วัตถุดิบ, ค่าแรง, ค่าใช้จ่ายอื่น, batch_quantity)` คำนวณใหม่ทุกครั้งที่ `product_costs` เปลี่ยน

---

## 9. ตัวอย่างโค้ด 1 feature (product)

```go
// internal/features/product/model.go
package product

import "strings"

// Product ข้อมูลสินค้า (ชื่อ json ต้องตรงกับ ProductModel ฝั่ง Flutter)
type Product struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Unit      string  `json:"unit"`
	CostPrice float64 `json:"cost_price"`
	SellPrice float64 `json:"sell_price"`
	Stock     int     `json:"stock"`
	MinStock  int     `json:"min_stock"`
}

// ProductRequest ข้อมูลที่รับมาตอนเพิ่ม / แก้สินค้า
type ProductRequest struct {
	Name      string  `json:"name"`
	Unit      string  `json:"unit"`
	CostPrice float64 `json:"cost_price"`
	SellPrice float64 `json:"sell_price"`
	MinStock  int     `json:"min_stock"`
}

// Validate เช็คข้อมูลก่อนบันทึก คืน error code ("" = ผ่าน)
func (req ProductRequest) Validate() string {
	if strings.TrimSpace(req.Name) == "" {
		return "product_name_required"
	}
	if req.CostPrice < 0 || req.SellPrice < 0 {
		return "product_price_invalid"
	}
	return ""
}
```

```go
// internal/features/product/repository.go
package product

// ErrNotFound ใช้บอก handler ว่าไม่เจอสินค้า (handler จะตอบ 404)
var ErrNotFound = errors.New("product not found")

// Repository คือรายการคำสั่งที่คุยกับ DB
// ทำเป็น interface เพื่อให้ตอน test ใช้ fakeRepository แทน DB จริงได้
type Repository interface {
	List(ctx context.Context) ([]Product, error)
	FindByID(ctx context.Context, id string) (Product, error)
	Create(ctx context.Context, req ProductRequest) (Product, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) List(ctx context.Context) ([]Product, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, unit, cost_price, sell_price, stock, min_stock
		FROM products
		ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []Product{} // ไม่ใช้ nil → JSON จะได้ [] แทน null
	for rows.Next() {
		var p Product
		err := rows.Scan(&p.ID, &p.Name, &p.Unit, &p.CostPrice, &p.SellPrice, &p.Stock, &p.MinStock)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (r *PostgresRepository) FindByID(ctx context.Context, id string) (Product, error) {
	var p Product
	err := r.db.QueryRow(ctx, `
		SELECT id, name, unit, cost_price, sell_price, stock, min_stock
		FROM products
		WHERE id = $1`, id).
		Scan(&p.ID, &p.Name, &p.Unit, &p.CostPrice, &p.SellPrice, &p.Stock, &p.MinStock)
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	return p, err
}

// Create ... เขียนแบบเดียวกัน (INSERT ... RETURNING ...)
```

```go
// internal/features/product/handler.go
package product

type Handler struct {
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// RegisterRoutes ต่อ route ของสินค้าเข้ากับกลุ่ม /api/v1
// router.go และ handler_test.go เรียกใช้ฟังก์ชันนี้ร่วมกัน → test ยิง path เดียวกับของจริง
func (h *Handler) RegisterRoutes(api gin.IRouter) {
	api.GET("/products", h.List)
	api.GET("/products/:id", h.Get)
	api.POST("/products", h.Create)
}

// List GET /api/v1/products
func (h *Handler) List(c *gin.Context) {
	products, err := h.repo.List(c.Request.Context())
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, products)
}

// Get GET /api/v1/products/:id
func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")

	product, err := h.repo.FindByID(c.Request.Context(), id)
	if errors.Is(err, ErrNotFound) {
		helpers.WriteError(c, http.StatusNotFound, "product_not_found")
		return
	}
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, product)
}

// Create POST /api/v1/products
func (h *Handler) Create(c *gin.Context) {
	var req ProductRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		helpers.WriteError(c, http.StatusBadRequest, "invalid_json")
		return
	}

	code := req.Validate()
	if code != "" {
		helpers.WriteError(c, http.StatusBadRequest, code)
		return
	}

	product, err := h.repo.Create(c.Request.Context(), req)
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusCreated, product)
}
```

```go
// internal/router/router.go — สร้าง router และต่อ route ของทุก feature
r := gin.Default() // มี Logger (พิมพ์ทุก request) + Recovery (โค้ด panic แล้ว server ไม่ล่ม) ให้แล้ว
r.Use(middleware.CORS(cfg.AllowedOrigin))
r.NoRoute(func(c *gin.Context) {
	helpers.WriteError(c, http.StatusNotFound, "not_found") // path ที่ไม่มี ก็ตอบ JSON แบบเดียวกับ error อื่น
})

api := r.Group("/api/v1")
api.GET("/health", func(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"}) // gin.H คือ map ไว้สร้าง JSON สั้น ๆ
})

productHandler := product.NewHandler(product.NewPostgresRepository(db))
productHandler.RegisterRoutes(api)
```

---

## 10. กฎทางธุรกิจ (สำคัญ)

**ออกใบกำกับภาษี** ทำใน DB transaction เดียว ขั้นไหนพัง ยกเลิกทั้งหมด:
1. รันเลขที่ใบกำกับ
2. บันทึกใบกำกับ + รายการสินค้า
3. ตัด stock ของทุกสินค้าในใบ
4. บันทึกรายรับ (ผูก `invoice_id`)

| เรื่อง | กฎ |
|---|---|
| คำนวณ VAT | คิดจากยอดรวมก่อน VAT **ทั้งใบ** (`utils.VAT(subtotal)`) ไม่คิดทีละบรรทัด |
| ตัด stock | `UPDATE products SET stock = stock - $1 WHERE id = $2 AND stock >= $1` ถ้าไม่มีแถวไหนถูกแก้ = stock ไม่พอ → 409 `stock_not_enough` |
| เลขที่ใบกำกับ | ค่าเริ่มต้น `INV-2026-0001` รันใหม่ทุกปี ใช้ `UPDATE invoice_numbers SET last_number = last_number + 1 WHERE year = $1 RETURNING last_number` ใน transaction → เลขไม่ซ้ำแม้กดพร้อมกัน |
| แก้ / ลบใบกำกับ | **ห้าม** ถ้าผิดให้ยกเลิก (`status = cancelled`) แล้วออกใบใหม่ |
| ยกเลิกใบกำกับ | คืน stock + ลบรายรับที่ผูกกับใบนั้น (ทำใน transaction เดียว) |
| ข้อมูลในใบ | เก็บชื่อ ที่อยู่ เลขผู้เสียภาษี ของผู้ขาย / ผู้ซื้อ และชื่อสินค้า **ณ วันที่ออก** ไว้ในใบเลย แก้ข้อมูลบริษัทหรือสินค้าภายหลัง ใบเก่าต้องไม่เปลี่ยน — **ยกเว้นโลโก้** ไม่เก็บซ้ำในใบ (กฎหมายไม่บังคับ + DB จะบวม) พิมพ์ใบเก่าซ้ำจะได้โลโก้ปัจจุบัน |
| รายรับจากใบกำกับ | แก้ / ลบตรง ๆ ไม่ได้ ต้องยกเลิกใบกำกับ |
| ลบสินค้า | ถ้าเคยอยู่ในใบกำกับแล้ว ลบไม่ได้ → 409 `product_in_use` |

---

## 11. Unit Test

- `go test ./...` ต้องรันได้ **โดยไม่ต้องมี DB** — handler test ใช้ `fakeRepository` แทน
- handler test สร้าง Gin router ใน test (`gin.TestMode`) + `RegisterRoutes` แล้วยิง request ด้วย `httptest`
- ไฟล์ test อยู่ข้างไฟล์จริง: `tax.go` → `tax_test.go`
- เขียนแบบ table-driven (ตาราง input → ผลที่ต้องได้)
- อะไรต้องมี test และแม่แบบ: ดู `.claude/skills/unit-test-check/SKILL.md`

---

## 12. การตั้งชื่อ

| อะไร | กฎ | ตัวอย่าง |
|---|---|---|
| package | ตัวเล็ก คำเดียว | `product`, `invoice` |
| ไฟล์ | `snake_case` | `handler.go`, `stock_movement.go` |
| ใช้นอก package | ขึ้นต้นตัวใหญ่ | `NewHandler`, `Product` |
| ใช้ใน package | ขึ้นต้นตัวเล็ก | `getEnv`, `health` |
| ตัวย่อ | ตัวใหญ่ทั้งคำ | `ID`, `VAT`, `URL`, `JSON` |
| error | `ErrXxx` | `ErrNotFound` |
| test | `TestXxx` ในไฟล์ `xxx_test.go` | `TestVAT` |
| error code ใน JSON | `snake_case` ภาษาอังกฤษ | `product_name_required` |

---

## 13. Config (environment variable)

| ชื่อ | ค่าเริ่มต้น | ใช้ทำอะไร |
|---|---|---|
| `PORT` | `8080` | port ที่ API เปิดรอ |
| `ALLOWED_ORIGIN` | `*` | เว็บที่อนุญาตให้เรียก API — ตอนพัฒนาใช้ `*` ได้ ขึ้น server จริงต้องใส่โดเมนเว็บ |
| `DATABASE_URL` | **ต้องมี** | ในเครื่อง: `postgres://manas@localhost:5432/costbook?sslmode=disable` |
| `JWT_SECRET` | **ต้องมี** | ข้อความลับไว้เซ็น token สร้างด้วย `openssl rand -hex 32` — แต่ละเครื่อง / server ใช้คนละค่า ถ้าเปลี่ยน token เก่าใช้ไม่ได้ทั้งหมด (ทุกคนต้อง login ใหม่) |
| `GIN_MODE` | `debug` | Gin อ่านค่านี้เอง ไม่ต้องเขียนใน config — ตอนพัฒนาไม่ต้องตั้ง (Gin จะพิมพ์ข้อความ debug ให้ดู) ขึ้น server จริงตั้งเป็น `release` |

ตอนพัฒนาเก็บค่าไว้ในไฟล์ `.env` (copy จาก `.env.example`) แล้วรันด้วย `make run` — Makefile โหลด `.env` ให้
เปลี่ยนค่าชั่วคราว: `make run PORT=9000`
รหัสผ่าน / secret ห้ามเขียนในโค้ด และห้าม commit ไฟล์ `.env` (อยู่ใน `.gitignore` แล้ว)

---

## 14. คำสั่งที่ใช้บ่อย

```bash
make run                                  # เปิด API ที่ http://localhost:8080 (โหลด .env ให้) กด Ctrl+C เพื่อปิด
make test                                 # รัน unit test ทั้งหมด
make check                                # gofmt + go vet + go test -cover
open http://localhost:8080/docs           # เปิดหน้าเอกสาร Swagger ใน browser
curl localhost:8080/api/v1/health         # ลองเรียก → {"status":"ok"}
curl -X POST localhost:8080/api/v1/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"admin@costbook.com","password":"1234"}'   # ลอง login → ได้ token
gofmt -l .                                # รายชื่อไฟล์ที่ยังไม่ได้จัดรูปแบบ (ว่าง = ดี)
gofmt -w .                                # จัดรูปแบบให้อัตโนมัติ
go vet ./...                              # ตรวจโค้ดที่น่าจะผิด
go test ./...                             # รัน unit test ทั้งหมด
go test -cover ./...                      # รัน test + ดู % coverage
go build -o bin/api ./cmd/api             # build เป็นไฟล์โปรแกรม
```

---

## 15. ต่อกับ Flutter (web-app-CostBook)

- ฝั่ง Flutter ตั้ง `ApiConstants.baseUrl = 'http://localhost:8080/api/v1'`
- ชื่อ field ใน JSON ต้องตรงกับ `fromJson` / `toJson` ของ model ฝั่ง Flutter
- error code จาก API → Flutter แสดงด้วย `'error.<code>'.tr` (เพิ่ม key `error.xxx` ใน `th.json` / `en.json`)
- login: เก็บ `token` ที่ได้ แล้วแนบ header `Authorization: Bearer <token>` ทุก request ที่ต้อง login — ได้ 401 `unauthorized` เมื่อไหร่ ให้ลบ token แล้วพาไปหน้า login
- PDF ใบกำกับ สร้างฝั่ง Flutter จากข้อมูล `GET /api/v1/invoices/{id}`
- แก้ API ที่กระทบ Flutter (เปลี่ยนชื่อ field / path) → แก้ทั้ง 2 repo และบันทึกใน `PROGRESS.md` ของทั้ง 2 ฝั่ง
