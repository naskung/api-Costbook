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
| `github.com/anthropics/anthropic-sdk-go/v2` | เรียก Claude (AI) — เช็คราคาตลาด + แชทผู้ช่วย (SDK ทางการของ Anthropic) | AI |

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
│   ├── middleware/           ทำงานก่อนทุก request: cors, auth (Authenticated = ต้อง login, RequireRole = เฉพาะบาง role) (logger มากับ gin.Default() แล้ว)
│   ├── helpers/              ช่วยงาน HTTP: WriteError, ServerError, MonthStart (?month=), UUIDParam (:id), ReadImage (อ่านรูปที่อัปโหลด) (ตอบสำเร็จใช้ c.JSON ของ Gin)
│   ├── utils/                ฟังก์ชันล้วน ๆ ไม่แตะ HTTP / DB: money (Round2, Profit), tax (VAT, PriceBeforeVAT), cost (CostPerUnit), date (IsDate), check (IsUUID, IsTaxID, IsEmail), token (JWT), image (ImageContentType), secret (Encrypt / Decrypt)
│   └── features/
│       ├── auth/             login + ข้อมูลผู้ใช้ที่ login อยู่ (ตาราง users) + ชื่อ role
│       ├── user/             โปรไฟล์ของตัวเอง (ชื่อ / รหัสผ่าน / รูป) + จัดการผู้ใช้ของร้าน (เฉพาะเจ้าของร้าน)
│       ├── company/          ข้อมูลร้าน + โลโก้ (ตาราง company มีแถวเดียว)
│       ├── product/
│       │   ├── model.go           struct + Validate()
│       │   ├── model_test.go
│       │   ├── repository.go      คุยกับ DB (interface + Postgres)
│       │   ├── handler.go         รับ request / ตอบ JSON + RegisterRoutes (ต่อ route ของ feature)
│       │   ├── handler_test.go    test ด้วย fakeRepository (ไม่ต้องมี DB)
│       │   ├── logic.go           (ถ้ามี) คำนวณเฉพาะ feature นี้
│       │   └── logic_test.go
│       ├── field/            ช่องข้อมูลเพิ่มเติมของสินค้าที่ร้านตั้งเอง (ตาราง product_fields)
│       ├── category/         หมวดหมู่สินค้า (ตาราง product_categories) สินค้าชี้หมวดด้วย category_id
│       ├── cost/             ใบต้นทุนของสินค้า (ตาราง product_costs) → คำนวณ cost_price + รายการที่ใช้บ่อย (cost_presets) + refresh.go (ราคาวัสดุเปลี่ยน → คิดต้นทุนสินค้าที่ใช้ใหม่)
│       ├── stock/            คงเหลือ / รับเข้า (คิดต้นทุนเฉลี่ย) / เบิกออก (ตาราง stock_movements) + materials.go (ตัด / คืนวัสดุตามใบต้นทุน)
│       ├── transaction/      รายรับ-รายจ่าย + สรุปรายเดือน
│       ├── invoice/          เอกสารขาย (ใบกำกับภาษี / ใบเสร็จ): ออกใบ (รันเลข + ตัด stock + บันทึกรายรับ) / ยกเลิก
│       ├── dashboard/        ตัวเลขสรุปหน้าแรก + กราฟ / กำไรขั้นต้น / ขายดี / เทียบตลาด (อ่านอย่างเดียว ไม่มีตารางของตัวเอง)
│       └── ai/               AI (Claude / Gemini ฟรี): เช็คราคาตลาด (ตาราง market_checks) + แชทผู้ช่วย + key ของร้าน (assistant.go, claude.go, gemini.go, prompt.go, tools.go, toolbox.go, market.go, chat.go, shop.go)
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
| 400 | ข้อมูลที่ส่งมาผิด | `invalid_json`, `month_invalid`, `email_required`, `password_required`, `company_name_required`, `company_tax_id_invalid`, `logo_required`, `logo_too_large`, `logo_type_invalid`, `product_name_required`, `product_price_invalid`, `product_min_stock_invalid`, `cost_batch_quantity_invalid`, `cost_type_invalid`, `cost_name_required`, `cost_amount_invalid`, `cost_material_invalid` (วัสดุที่ผูกไม่มีจริง / เป็นสินค้าตัวเอง / บรรทัดแบบ %), `stock_product_required`, `stock_quantity_invalid`, `transaction_type_invalid`, `transaction_category_required`, `transaction_amount_invalid`, `transaction_date_invalid`, `invoice_date_invalid`, `invoice_customer_required`, `invoice_customer_tax_id_invalid`, `invoice_items_required`, `invoice_item_invalid`, `product_image_required`, `product_image_too_large`, `product_image_type_invalid`, `ai_key_format_invalid`, `ai_key_invalid` (ตอนบันทึก key แล้ว Anthropic ไม่รับ) |
| 401 | ยังไม่ login / token ผิดหรือหมดอายุ / login ไม่ผ่าน | `unauthorized`, `invalid_credentials` |
| 404 | ไม่เจอข้อมูล / ไม่มี path นี้ / id ไม่ใช่ UUID | `product_not_found`, `transaction_not_found`, `invoice_not_found`, `logo_not_found`, `product_image_not_found`, `not_found` (จาก `r.NoRoute` ใน router) |
| 409 | ขัดกับข้อมูลที่มีอยู่ | `stock_not_enough`, `material_not_enough` (ตอบ field เพิ่ม `material` / `unit` / `need` / `have` ให้หน้าเว็บโชว์), `product_not_for_sale`, `product_in_use`, `transaction_from_invoice`, `invoice_already_cancelled`, `company_tax_id_required`, `ai_search_unavailable` (AI ฟรีค้นเว็บไม่ได้ → เช็คราคาตลาดไม่ได้) |
| 422 | AI ปฏิเสธคำขอ (นโยบายความปลอดภัย) | `ai_refused` |
| 500 | server พัง | `server_error` (ใช้ `helpers.ServerError` → error จริงลง log ไม่ส่งให้ client) |
| 502 | AI ตอบไม่ครบ / เรียก Claude ไม่สำเร็จ | `ai_no_result`, `ai_error` (รายละเอียดลง log) |
| 503 | ไม่มี AI ให้ใช้ / key ผิด / เครดิตหมด / โควตาฟรีหมด / AI ไม่ว่าง | `ai_not_configured`, `ai_key_invalid`, `ai_credit_low` (เครดิตของ key Claude หมด — Anthropic ตอบ 400 "credit balance is too low" หรือ `billing_error`), `ai_quota_exceeded` (Gemini ฟรีใช้เกินโควตา), `ai_busy` |
| 504 | AI ใช้เวลานานเกิน | `ai_timeout` |

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
| product ✅ | `GET /api/v1/products` | รายการสินค้า |
| | `GET /api/v1/products/{id}` | สินค้า 1 ตัว |
| | `POST /api/v1/products` | เพิ่มสินค้า |
| | `PUT /api/v1/products/{id}` | แก้สินค้า |
| | `DELETE /api/v1/products/{id}` | ลบสินค้า |
| | `GET /api/v1/products/{id}/image?v=<image_version>` | รูปสินค้า (ตอบเป็นรูป, ต้อง login) |
| | `PUT /api/v1/products/{id}/image` | อัปโหลดรูปสินค้า (multipart ช่อง `image`, PNG / JPG / WebP ไม่เกิน 1 MB) → ได้สินค้าที่มี `image_version` ใหม่ |
| | `DELETE /api/v1/products/{id}/image` | ลบรูปสินค้า → ได้สินค้าที่ `image_version = 0` |
| field ✅ | `GET /api/v1/product-fields` | ช่องข้อมูลเพิ่มเติมของสินค้าที่ร้านตั้งเอง (เช่น วัสดุ สี ขนาด) |
| | `POST /api/v1/product-fields` | เพิ่มช่อง |
| | `PUT /api/v1/product-fields/{id}` | แก้ช่อง |
| | `DELETE /api/v1/product-fields/{id}` | ลบช่อง (ค่าของช่องนี้ในทุกสินค้าหายไปด้วย) |
| category ✅ | `GET /api/v1/product-categories` | หมวดหมู่สินค้า (เรียงตามชื่อ) + จำนวนสินค้าในแต่ละหมวด |
| | `POST /api/v1/product-categories` | เพิ่มหมวด (ชื่อห้ามซ้ำ ไม่สนตัวพิมพ์เล็ก / ใหญ่ → 409 `category_name_taken`) |
| | `PUT /api/v1/product-categories/{id}` | เปลี่ยนชื่อหมวด → สินค้าทุกตัวในหมวดเห็นชื่อใหม่ทันที |
| | `DELETE /api/v1/product-categories/{id}` | ลบหมวด → สินค้าในหมวดกลายเป็นไม่มีหมวด (ไม่ลบสินค้า) |
| cost ✅ | `GET /api/v1/products/{id}/costs` | ใบต้นทุน: รายการต้นทุน (วัตถุดิบ / ค่าแรง / ค่าใช้จ่ายอื่น) + จำนวนผลิตต่อรอบ + ต้นทุน / ราคาขายต่อชิ้น |
| | `PUT /api/v1/products/{id}/costs` | บันทึกทั้งใบ (แทนที่รายการเดิมทั้งหมด) → คำนวณ `cost_price` ใหม่ + บันทึก `sell_price` |
| | `GET /api/v1/cost-presets` | รายการต้นทุนที่ใช้บ่อย (เช่น เส้น PLA กรัมละ 0.45) ไว้เลือกใส่ใบต้นทุนเร็ว ๆ |
| | `POST /api/v1/cost-presets` | เพิ่มรายการที่ใช้บ่อย |
| | `PUT /api/v1/cost-presets/{id}` | แก้ |
| | `DELETE /api/v1/cost-presets/{id}` | ลบ |
| stock ✅ | `GET /api/v1/stock?low=true` | จำนวนคงเหลือ (`low=true` = เฉพาะใกล้หมด) |
| | `GET /api/v1/stock/movements?product_id=` | ประวัติรับเข้า / เบิกออก |
| | `POST /api/v1/stock/in` | รับสินค้าเข้า |
| | `POST /api/v1/stock/out` | เบิกสินค้าออก |
| transaction ✅ | `GET /api/v1/transactions?month=2026-10` | รายรับ-รายจ่ายของเดือน |
| | `GET /api/v1/transactions/summary?month=2026-10` | สรุปเงินของเดือน: รายได้ / ต้นทุนสินค้าที่ขาย / กำไรขั้นต้น / รายจ่าย / กำไรสุทธิ (ดูกฎ "กำไร" ในหัวข้อ 10) |
| | `POST /api/v1/transactions` | เพิ่มรายการ |
| | `PUT /api/v1/transactions/{id}` | แก้รายการ |
| | `DELETE /api/v1/transactions/{id}` | ลบรายการ |
| invoice ✅ | `GET /api/v1/invoices?month=2026-10` | เอกสารขายของเดือน (ใบกำกับภาษี + ใบเสร็จรับเงิน) |
| | `GET /api/v1/invoices/{id}` | เอกสาร 1 ใบ (Flutter เอาไปสร้าง PDF) |
| | `POST /api/v1/invoices` | ออกเอกสารขาย — ร้านจด VAT ได้ใบกำกับภาษี ไม่จด ได้ใบเสร็จ |
| | `POST /api/v1/invoices/{id}/cancel` | ยกเลิกเอกสาร |
| dashboard ✅ | `GET /api/v1/dashboard?month=2026-10` | ตัวเลขสรุปหน้าแรก + ยอดขายรายวัน / ย้อนหลัง 6 เดือน / กำไรขั้นต้น / ขายดี / ตามหมวด / มูลค่าสต็อก / เทียบราคาตลาด |
| ai ✅ | `GET /api/v1/ai/status` | ตอนนี้ใช้ AI ตัวไหน (Claude / Gemini ฟรี / ไม่มี) ค้นเว็บได้ไหม ร้านใส่ key เองไหม |
| | `PUT /api/v1/ai/key` | ร้านใส่ key ของ Claude เอง (ลอง key กับ Anthropic ก่อน แล้วเก็บแบบเข้ารหัส) |
| | `DELETE /api/v1/ai/key` | ลบ key ของร้าน → กลับไปใช้ key ใน `.env` หรือ AI ฟรี |
| | `POST /api/v1/ai/chat` | แชทกับผู้ช่วย AI (ส่งประวัติแชททั้งหมด ได้คำตอบ + งานที่ AI เสนอ) |
| | `GET /api/v1/products/{id}/market` | ผลเช็คราคาตลาดครั้งล่าสุด (ไม่เสียเงิน) |
| | `POST /api/v1/products/{id}/market` | ให้ AI ค้นราคาตลาดใหม่แล้วบันทึกผล (มีค่าใช้จ่าย 1–3 นาที) |
| auth ✅ | `POST /api/v1/auth/login` | login ได้ token |
| | `GET /api/v1/auth/me` | ข้อมูลผู้ใช้ที่ login อยู่ (ต้องส่ง token) |
| user ✅ | `PUT /api/v1/profile` | แก้ชื่อ-นามสกุลของตัวเอง |
| | `PUT /api/v1/profile/password` | เปลี่ยนรหัสผ่านของตัวเอง (ต้องใส่รหัสเดิมถูก) → 204 |
| | `PUT /api/v1/profile/avatar` | อัปโหลดรูปโปรไฟล์ (multipart ช่อง `avatar`, PNG / JPG / WebP ไม่เกิน 1 MB) |
| | `DELETE /api/v1/profile/avatar` | ลบรูปโปรไฟล์ → ได้ผู้ใช้ที่ `avatar_version = 0` |
| | `GET /api/v1/users/{id}/avatar?v=<avatar_version>` | รูปโปรไฟล์ (ตอบเป็นรูป ทุกคนที่ login ดูได้) |
| | `GET /api/v1/users` | รายชื่อผู้ใช้ทั้งร้าน — **เฉพาะเจ้าของร้าน** |
| | `POST /api/v1/users` | เพิ่มผู้ใช้ (อีเมล + รหัสผ่านแรก + role) — **เฉพาะเจ้าของร้าน** |
| | `PUT /api/v1/users/{id}` | แก้ชื่อ / role / เปิด-ปิดบัญชี — **เฉพาะเจ้าของร้าน** |
| | `PUT /api/v1/users/{id}/password` | ตั้งรหัสผ่านใหม่ให้ (ไม่ต้องรู้รหัสเดิม) → 204 — **เฉพาะเจ้าของร้าน** |

ในตารางเขียน path แบบ `{id}` แต่ในโค้ด Gin เขียนเป็น `:id` เช่น `api.GET("/products/:id", ...)` แล้วอ่านค่าด้วย `helpers.UUIDParam(c, "id", "product_not_found")` (id ไม่ใช่ UUID → ตอบ 404 ให้เลย ไม่ส่งไป DB)

ทุก endpoint ยกเว้น health / auth/login / GET company / GET company/logo **ต้อง login** (`authRequired`)
endpoint ที่เขียนว่า **เฉพาะเจ้าของร้าน** → role อื่นได้ 403 `forbidden` (ตอนนี้มีแค่จัดการผู้ใช้ ที่เหลือทุก role ใช้ได้เหมือนกัน)
`?month=2026-10` ไม่ส่ง = เดือนนี้ อ่านด้วย `helpers.MonthStart(c)` ได้วันแรกของเดือน `"2026-10-01"` ไปใช้ใน SQL: `date >= $1::date AND date < $1::date + interval '1 month'`
รายละเอียด field ของทุก endpoint ดูใน Swagger (`docs/openapi.yaml`)

### รายละเอียด auth (ทำแล้ว)

`POST /api/v1/auth/login`

```json
// request
{ "email": "admin@costbook.com", "password": "1234" }

// 200
{
  "token": "eyJhbGciOi...",
  "user": { "id": "a00d265d-...", "email": "admin@costbook.com", "first_name": "Admin", "last_name": "CostBook", "role": "owner", "avatar_version": 0 }
}
```

- อีเมลไม่สนตัวพิมพ์เล็ก / ใหญ่ และตัดช่องว่างหน้า-หลังให้ (เก็บใน DB เป็นตัวเล็กเสมอ)
- error: 400 `invalid_json` / `email_required` / `password_required`, 401 `invalid_credentials` (อีเมลไม่มี หรือรหัสผิด ตอบเหมือนกัน), 403 `user_disabled` (รหัสถูก แต่เจ้าของร้านปิดบัญชีไว้ — รหัสผิดยังได้ 401 เสมอ ไม่บอกว่าบัญชีถูกปิด)

`GET /api/v1/auth/me` — header `Authorization: Bearer <token>`

- 200 `{ "id", "email", "first_name", "last_name", "role", "avatar_version" }`
- 401 `unauthorized` (ไม่มี token / token ผิด / หมดอายุ / ผู้ใช้ถูกลบ / ถูกปิดบัญชี) → Flutter พาไปหน้า login

route ใหม่ที่ต้อง login ก่อน → ใส่ `authRequired` ใน `router.go` แล้ว handler อ่าน id ผู้ใช้ด้วย `c.GetString(middleware.UserIDKey)` และ role ด้วย `c.GetString(middleware.RoleKey)`
- `authRequired` = `middleware.Authenticated(jwtSecret, authRepo.Status)`: ตรวจ token แล้ว **ถามสถานะผู้ใช้จาก DB ทุก request** → ผู้ใช้ถูกลบ / ถูกปิดบัญชี ใช้ token เดิมต่อไม่ได้ทันที (ไม่ต้องรอ token หมดอายุ 7 วัน) + ใส่ role ไว้ใน context
- จำกัดเฉพาะบาง role → ต่อ `middleware.RequireRole(auth.RoleOwner, ...)` หลัง `authRequired` เช่น `api.GET("/users", authRequired, middleware.RequireRole(auth.RoleOwner), h.List)` → role อื่นได้ 403 `forbidden`
- `middleware.Auth` (ตรวจแค่ token ไม่ถาม DB) ยังอยู่ ใช้ใน test ของ feature อื่น

### รายละเอียด ผู้ใช้ + role (ทำแล้ว, `000014_user_roles`)

- role มี 3 แบบ (ค่าคงที่ใน `auth/model.go`): `owner` = เจ้าของร้าน, `manager` = ผู้จัดการ, `staff` = พนักงาน — ผู้ใช้เดิมก่อนมี role เป็น `owner` ทั้งหมด
- **ตอนนี้จำกัดสิทธิ์แค่เรื่องเดียว:** จัดการผู้ใช้ (`/users`) เฉพาะ `owner` / เมนูอื่นทุก role ใช้ได้เหมือนกัน — ทำโครงไว้แล้ว (`RequireRole`) รอผู้ใช้กำหนดว่า role ไหนเห็นเมนูไหน / ทำอะไรได้
- `active = false` = ปิดบัญชี: login ไม่ได้ (403 `user_disabled`) และ token เดิมใช้ไม่ได้ทันที (401) / ข้อมูลที่เคยบันทึกยังอยู่ (ไม่มีลบผู้ใช้ ใช้ปิดบัญชีแทน)
- **ร้านต้องมีเจ้าของที่ใช้งานได้อย่างน้อย 1 คน:** เปลี่ยน role / ปิดบัญชีเจ้าของคนสุดท้าย → 409 `user_last_owner` (ล็อกแถวเจ้าของทุกคน `FOR UPDATE` ก่อนนับ กันเจ้าของ 2 คนปลดกันพร้อมกันจนไม่เหลือใคร — `user/repository.go` → `Update`)
- รหัสผ่านอย่างน้อย 6 ตัวอักษร (นับเป็นตัวอักษร ภาษาไทย 1 ตัว = 1) เก็บแบบ bcrypt / ไม่มี `password_hash` ออกไปใน JSON เด็ดขาด
- อีเมลเก็บเป็นตัวเล็ก ตัดช่องว่าง / ซ้ำ → 409 `user_email_taken` / เจ้าของร้านแก้อีเมลไม่ได้ (ใช้ login)
- รูปโปรไฟล์เก็บใน DB แบบเดียวกับรูปสินค้า: `avatar_version` = เวลาที่เปลี่ยนรูป (มิลลิวินาที) ใส่ใน URL `?v=` → browser โหลดรูปใหม่ / 0 = ไม่มีรูป
- error อื่น: 400 `user_email_invalid` / `user_name_required` / `user_name_too_long` / `user_role_invalid` / `password_too_short` / `password_current_required` / `password_current_invalid` / `avatar_required` / `avatar_too_large` / `avatar_type_invalid`, 404 `user_not_found` / `avatar_not_found`

### รายละเอียด product / cost / stock (ทำแล้ว)

- สินค้าใหม่ `stock = 0` — เพิ่มของผ่าน `POST /stock/in` เท่านั้น (`PUT /products/{id}` แก้ stock ไม่ได้)
- ลบสินค้าที่เคยอยู่ในใบกำกับไม่ได้ → 409 `product_in_use` / ลบได้ → รายการต้นทุนและประวัติ stock หายไปด้วย
- สินค้ามี `code` (รหัส / บาร์โค้ด ว่างได้ ถ้าใส่ห้ามซ้ำ → 409 `product_code_taken`), `category_id` (หมวด ว่าง = ไม่มี, id ไม่มีจริง → 400 `product_category_invalid`) + `category` (ชื่อหมวด อ่านอย่างเดียว ได้จาก JOIN), `description`, `track_stock`, `custom_fields`
- อ่าน / บันทึกสินค้าใช้ `products p LEFT JOIN product_categories c` — INSERT / UPDATE เขียนเป็น CTE `WITH p AS (... RETURNING *)` แล้ว SELECT พร้อมชื่อหมวด (`selectSaved`)
- AI เสนอสินค้าด้วยชื่อหมวด → `toolbox.categoryID` หา id ให้ (ไม่มีหมวดชื่อนี้ = บอก AI ให้เลือกหมวดที่มี หรือให้ผู้ใช้เพิ่มหมวดก่อน AI สร้างหมวดเองไม่ได้)
- `track_stock = false` (งานบริการ / งานสั่งทำ) → ไม่โชว์ในหน้าสต็อก ไม่เตือนใกล้หมด ขายได้ไม่จำกัด รับเข้า / เบิกออกไม่ได้ (409 `stock_not_tracked`)
- **รูปสินค้า** 1 รูป (`000010_product_image`): เก็บใน DB แบบเดียวกับโลโก้ (ตรวจไฟล์ด้วย `helpers.ReadImage` ตัวเดียวกัน) / `image_version` = เวลาที่เปลี่ยนรูปล่าสุด (มิลลิวินาที, 0 = ไม่มีรูป) — `GET /products` ไม่ส่งตัวรูปมา ส่งแค่ `image_version`
  - Flutter โหลดรูปที่ `/products/{id}/image?v=<image_version>` พร้อม token → เปลี่ยนรูปแล้ว URL เปลี่ยน จึงให้ browser เก็บรูปไว้ได้นาน (`Cache-Control: private, max-age=31536000, immutable`)
  - เปลี่ยนรูปผ่าน `PUT` / `DELETE /products/{id}/image` เท่านั้น (`PUT /products/{id}` ไม่แตะรูป)
- `PUT /products/{id}` **ไม่ส่ง** `track_stock` / `custom_fields` มา = ใช้ค่าเดิม (SQL `COALESCE($n, column)`) — แอปเก่าที่ไม่รู้จัก field ใหม่จะไม่ล้างค่าทิ้ง
- ใบต้นทุน `PUT /products/{id}/costs` ส่ง `{ "batch_quantity", "sell_price", "items": [...] }` ทั้งใบ แต่ละบรรทัดมี 2 แบบ (`kind`)
  - `fixed` = `quantity × unit_price` เช่น เส้น PLA 160 กรัม × 0.45 = 72
  - `percent` = `percent`% ของผลรวมบรรทัดแบบ `fixed` ทั้งใบ เช่น เผื่อพิมพ์เสีย 10% (0–1000 ไม่งั้น 400 `cost_percent_invalid`)
  - API คิด `amount` ของทุกบรรทัดเอง (`cost/logic.go`) แล้ว `cost_price = ผลรวม ÷ batch_quantity` บันทึกลงสินค้า รายการเรียงตามประเภท แล้วตามลำดับที่กรอก (`line_no`)
- รายการที่ใช้บ่อย (`cost_presets`) = แม่แบบบรรทัดต้นทุน `{ "type", "kind", "name", "unit", "unit_price", "percent" }` ไม่เกี่ยวกับสินค้าตัวไหน
- รับเข้า / เบิกออก `{ "product_id", "quantity", "note", "unit_cost", "supplier" }` ใช้ SQL `stock = stock + $1 WHERE stock + $1 >= 0` → เบิกเกินที่มีได้ 409 `stock_not_enough`
- รับเข้าแล้วใส่ `unit_cost` > 0 → `cost_price` ของสินค้า = ต้นทุนเฉลี่ย `(ของเดิม × ทุนเดิม + ของใหม่ × ทุนใหม่) ÷ จำนวนรวม` (ร้านซื้อมาขายไป ไม่ต้องใช้หน้าคำนวณต้นทุน) — ล็อกแถวสินค้าด้วย `SELECT ... FOR UPDATE` กันรับเข้าพร้อมกันแล้วคิดผิด
- ประวัติที่มี `invoice_no` = เกิดจากใบกำกับ (out = ออกใบ, in = ยกเลิกใบ)

### รายละเอียด วัสดุในสต็อก (ทำแล้ว, `000012_materials`)

- **วัสดุ** = สินค้าที่ `for_sale = false` (เช่น เส้น PLA เป็นกรัม, ห่วงพวงกุญแจ) — นับสต็อก / รับเข้าใส่ทุนได้เหมือนสินค้าปกติ แต่ **ขายไม่ได้** (ใส่ในเอกสารขาย → 409 `product_not_for_sale`) / ไม่ส่ง `for_sale` = ขาย (`*bool` + `COALESCE`)
- บรรทัดต้นทุนแบบ `fixed` ผูกวัสดุได้ด้วย `material_id` → API ใช้ **ทุนของวัสดุ** (`cost_price`) + หน่วยของวัสดุเป็น `unit_price` / `unit` เอง ไม่ใช้ค่าที่ส่งมา (`ResolveMaterials` ใน cost) / ผูกตัวเอง หรือวัสดุไม่มีจริง → 400 `cost_material_invalid` / ลบวัสดุ → บรรทัดยังอยู่ แค่หลุดการผูก (`ON DELETE SET NULL`)
- **ตัดวัสดุอัตโนมัติ** (`stock.UseMaterials` ใน transaction เดียวกัน) — ผู้ใช้เลือก 2026-11-01:
  - รับสินค้าที่ผลิตเสร็จเข้าสต็อก (`POST /stock/in`) → ตัดวัสดุทุกตัวในใบต้นทุนที่นับสต็อก จำนวนที่ตัด = `round(จำนวนต่อรอบ ÷ batch_quantity × จำนวนที่รับเข้า)` (`stock/logic.go` → `materialNeed`) เช่น พวงกุญแจ 6 ชิ้นใช้ PLA 60 กรัม รับเข้า 12 ชิ้น → ตัด 120 กรัม
  - ส่ง `skip_materials: true` = ไม่ตัด (ของที่ทำไว้ก่อนแล้ว / ซื้อมา)
  - สินค้าที่ไม่นับสต็อก (`track_stock = false` เช่น รับพิมพ์ตามไฟล์) → ตัดวัสดุตอน **ขาย** แทน / ยกเลิกเอกสาร → คืนวัสดุ (`stock.ReturnMaterials` รวมจากแถว `is_material` ของเอกสารนั้น)
  - วัสดุไม่พอ → 409 `material_not_enough` ไม่บันทึกอะไรเลย ตอบวัสดุตัวแรกที่ไม่พอ (เรียงตามชื่อ) `{ "error", "material", "unit", "need", "have" }` (`stock.WriteShortage`)
  - ประวัติที่ตัด / คืนวัสดุมี `is_material = true` หมายเหตุ `ใช้ผลิต {สินค้า} × {จำนวน}` / `คืนวัสดุ (ยกเลิกเอกสาร)` — ล็อกแถววัสดุ `FOR UPDATE` ก่อนเช็ค
- **ราคาวัสดุเปลี่ยน → ต้นทุนสินค้าที่ใช้เปลี่ยนตาม** (`cost.RefreshDependents`): เรียกหลังรับเข้าที่ทำให้ทุนเฉลี่ยเปลี่ยน / แก้สินค้า / บันทึกใบต้นทุน — ไล่หาสินค้าที่ผูกวัสดุนี้ (BFS + กันวนซ้ำ เผื่อวัสดุทำจากวัสดุอีกที) แล้วคิด `amount` + `cost_price` ใหม่ / `cost.DB` เป็น interface ที่ทั้ง pool และ tx ใช้ได้
- `cost_price` มีทศนิยม 2 ตำแหน่ง → ทุนต่อกรัมถูกปัด (เช่น 0.459 → 0.46) / จำนวนที่ตัดเป็นจำนวนเต็ม (stock เป็น `INTEGER`)

### รายละเอียด ภาพรวมแบบละเอียด + AI (ทำแล้ว)

**ภาพรวม** (`GET /dashboard`) เพิ่ม `sales`, `previous`, `monthly_trend`, `daily_sales`, `previous_daily_sales`, `top_products`, `category_sales`, `expense_by_category`, `stock`, `market` (field เดิมยังอยู่ แอปเก่าใช้ได้)
- แต่ละเดือนใน `monthly_trend` / `previous` มี `bills` (จำนวนเอกสารขาย) และ `gross_profit` ด้วย → หน้าเว็บคิด % เทียบเดือนก่อนได้ทุกตัว
- `previous_daily_sales` = ยอดขายรายวันของเดือนก่อน (ไว้วาดเส้นเทียบ) — เดือนก่อนคิดด้วย `previousMonthStart` (ม.ค. → ธ.ค. ปีก่อน)

- ยอดขาย = **ก่อน VAT หลังหักส่วนลด** (`invoices.subtotal`) — เงินที่ร้านได้จริง ใช้คิดกำไร
- ยอดของแต่ละบรรทัด (สินค้าขายดี / ตามหมวด) = ยอดบรรทัด × (`subtotal` ÷ `items_total`) → แบ่งส่วนลดและ VAT ตามสัดส่วน
- ต้นทุนของที่ขาย = จำนวน × `invoice_items.cost_price` (ต้นทุน **ณ วันที่ขาย** เก็บไว้ตอนออกใบ — แก้ต้นทุนสินค้าทีหลัง กำไรเดือนเก่าไม่เปลี่ยน)
- กำไรขั้นต้น = ยอดขาย − ต้นทุนของที่ขาย, margin = กำไร ÷ ยอดขาย × 100 (`dashboard/logic.go`)
- เทียบราคาตลาด: ใช้ราคาที่ลูกค้าจ่าย (ร้านจด VAT ที่ตั้งราคาไม่รวม VAT → บวก VAT ก่อน) เทียบกับช่วงราคาปลีกจากการเช็คครั้งล่าสุด → `below` / `within` / `above` / `unknown`
- % เทียบเดือนก่อน Flutter คิดเอง จาก `previous`

**AI** (`features/ai`) มี 2 ตัว เลือกให้เองทุกคำขอ (`pickProvider` ใน `ai/assistant.go`) ตามลำดับ:

1. key ของ Claude ที่ร้านใส่ในหน้าตั้งค่า (`company.ai_key` เข้ารหัสไว้)
2. `ANTHROPIC_API_KEY` ใน `.env`
3. `GEMINI_API_KEY` ใน `.env` → **Gemini ฟรี** (ค่าเริ่มต้นเมื่อร้านไม่มี key)
4. ไม่มีเลย → ปิด AI (ตอบ 503 `ai_not_configured`) ส่วนอื่นใช้ได้ปกติ

| | Claude (`claude-opus-5-5`, SDK ทางการ `ai/claude.go`) | Gemini ฟรี (`gemini-3.8-flash`, REST `ai/gemini.go`) |
|---|---|---|
| แชท + อ่านข้อมูลร้าน + เสนองาน | ได้ | ได้ (เครื่องมือชุดเดียวกัน) |
| ค้นเว็บ → เช็คราคาตลาด | ได้ | **ได้เมื่อมี `TAVILY_API_KEY`** (ค้นเว็บฟรี 1,000 ครั้ง/เดือน ไม่ต้องผูกบัตร) / หรือ `GEMINI_WEB_SEARCH=on` เมื่อเปิด Gemini แบบเสียเงิน / ไม่มีทั้งคู่ → 409 `ai_search_unavailable` (ในแชท AI จะบอกวิธีเปิด) |
| ค่าใช้จ่าย | จ่ายตามการใช้งาน | ฟรี มีโควตาต่อวัน (เกิน → 503 `ai_quota_exceeded`) — Google อาจเอาข้อความไปพัฒนาบริการ |

- `Assistant` interface (`Chat`, `Research`) → handler ไม่ต้องรู้ว่าเป็น AI ตัวไหน
- เครื่องมือเขียนครั้งเดียว (`toolSpec` ใน `ai/tools.go`) แล้วแปลงเป็นรูปแบบของแต่ละตัว: `claudeTools` / `geminiTools`
- **Gemini สลับรุ่นเอง** (`ai/gemini_fallback.go`): ทุกคำขอไป Gemini ผ่าน `fallbackGemini` — รุ่นหลัก (`GEMINI_MODEL`) ตอบ 5xx (ยุ่ง) / 429 (โควตาของรุ่นนั้นเต็ม — ฟรีนับแยกรายรุ่น) / เงียบเกิน 60 วินาที (`geminiAttemptTimeout`) → ลองรุ่นสำรองถัดไป (`GEMINI_FALLBACK_MODELS` ไม่ใส่ = `gemini-3.5-flash`, `gemini-3.5-flash-lite`), 4xx อื่นไม่ลองต่อ, สลับแล้วใช้รุ่นนั้นต่อทั้งงาน (คำตอบของ Gemini ผูกกับรุ่น) / ลองจริงแล้ว: 3.6 Flash 503 → 3.5 Flash 429 → 3.5 Flash-Lite ตอบได้
- **log สาเหตุ**: ทุกครั้งที่ AI ใช้ไม่ได้ เขียน `WARN AI ใช้ไม่ได้ code=... detail=...` (`writeAICode`) — ดูในหน้าต่าง `make run` ได้ว่า 503 เพราะอะไร
- **Gemini ฟรี + Tavily** (`ai/tavily.go`, `geminiSearchResearch` ใน `ai/gemini_market.go`) — วิธีเช็คราคาตลาดแบบฟรี (ผู้ใช้เลือก 2026-11-01):
  1. Gemini คิดคำค้น 2–3 คำ **+ เลือกแหล่ง** ของแต่ละคำ (JSON `geminiQuerySchema`, `cleanQueries` ตัดซ้ำ / ว่าง): `marketplace` = ร้านค้าในไทยทุกแหล่ง (ของทั่วไป งานทำมือ ของสำเร็จรูป) / `central_market` = ตลาดกลาง / ราคาทางการ (ของสด: เนื้อสัตว์ ผัก ผลไม้ อาหารทะเล ไข่ ข้าว) — ผู้ใช้เลือก 2026-11-01 "ให้เช็คจาก Shopee Lazada เป็นหลัก ของสดเช็คจากตลาดกลาง" แล้วเปลี่ยนเป็น "เช็คราคาในไทย ไม่จำเป็นต้อง Shopee Lazada อย่างเดียว"
  2. `searchPlan` (`ai/search_plan.go`) แปลงเป็นการค้นจริง ไม่เกิน `maxSearches` = 6 ครั้ง: ร้านค้า → **ทุกคำค้น** ค้นทั้งเว็บไทย 1 ครั้ง (ไม่จำกัดเว็บ ป้าย `[ร้านค้าในไทย]`) / **คำค้นแรก** (ชื่อสินค้า) ค้น `shopee.co.th` 1 + `lazada.co.th` 1 แยกเพิ่ม (ร้านเยอะ ราคาครบ / ค้นรวมกันผลเป็น Shopee หมด) แล้วการค้นทั้งเว็บของคำนั้นใส่ `exclude_domains` 2 เว็บนี้ จะได้เจอร้านแบบอื่น (`shopSearches`) / ตลาดกลาง → 1 ครั้ง `include_domains` = ตลาดไท ตลาดสี่มุมเมือง dit / moc / oae แบบ `prefer` (เว็บราคาเกษตรอื่นก็ได้) + `time_range: month` (ข่าวราคาเก่าใช้ไม่ได้) — Tavily `POST /search` (`search_depth: basic` = 1 เครดิต, `max_results: 8`, `country: thailand`, key ทาง header) ผลแต่ละหน้ามีป้าย `[ร้านค้าในไทย]` / `[Shopee]` / `[Lazada]` / `[ตลาดกลาง / ราคาทางการ]` (`SearchResult.Source`)
  3. Gemini จัดเป็นผล (`geminiWriteReport`) — `marketRules` บอกให้ใช้ราคาจากร้านค้าในไทยทุกแหล่ง (Shopee, Lazada, เว็บร้านค้า, ห้าง, Facebook) เฉพาะราคาเงินบาทของร้านในไทย ไม่เอาเว็บต่างประเทศ / ของสดใช้ราคาตลาดกลางล่าสุด (Claude ใช้กติกาเดียวกัน) แหล่งที่มาเฉพาะลิงก์ที่ค้นเจอจริง — เช็ค 1 สินค้า ≈ 5–6 เครดิต → ฟรีประมาณ 170 สินค้า / เดือน
  - ลองจริงแล้ว: พวงกุญแจ 3D → Shopee 2 + Lazada 1 ร้าน 45–153.90 บาท / หลังเปลี่ยนเป็นทั้งเว็บไทย → Lazada + เว็บร้านค้าไทย (ptcargomember.com) 96–178 บาท / หมูสามชั้น → พาณิชย์จังหวัด + เว็บราคาหมู + Lazada 150–200 บาท/กก. (หน้า Shopee ที่ Tavily อ่านได้ ราวครึ่งหนึ่งมีราคาให้เห็น)
  - error ของ Tavily (`TavilyError`): 401 / 403 → 503 `search_key_invalid`, 429 / 432 / 433 (เครดิตเดือนนี้หมด) → 503 `search_quota_exceeded`, อื่น ๆ → log + 502 `ai_error` / เครดิตหมดกลางทาง → หยุดค้นทันที
  - ลำดับ: `GEMINI_WEB_SEARCH=on` → ค้นผ่าน Google ก่อน / ไม่งั้นมี `TAVILY_API_KEY` → Tavily / ไม่มีทั้งคู่ → ค้นไม่ได้
- **Gemini ค้น Google** (`geminiResearch` ใน `ai/gemini_market.go`, ปิดอยู่เป็นค่าเริ่มต้น): ลองกับบัญชีฟรีจริงแล้ว (2026-11-01) ค้นไม่ได้ — รุ่น 2.5 (ที่หน้าราคาบอกว่าค้นฟรีวันละ 500 ครั้ง) ตอบ 404 "no longer available to new users", รุ่น 3.x ใส่ `google_search` แล้วได้ 429 ทันที (โควตาฟรี = 0) ส่วนแชทธรรมดายังใช้ได้ / แบบเสียเงิน: ค้นฟรี 5,000 ครั้ง/เดือน (Gemini 3 ขึ้นไป) + 3.8 Flash $0.75 / $3.75 ต่อล้าน token (ถึง 31 ธ.ค. 2026) → เช็ค 1 ครั้งไม่ถึง 1 บาท และ Google ไม่เอาข้อมูลไปใช้
  - ทำ 2 ขั้น (ค้น Google พร้อมตอบเป็น JSON ครั้งเดียวไม่ได้): 1) รุ่นค้น (`GEMINI_SEARCH_MODEL` ไม่ใส่ = รุ่นเดียวกับแชท) + `tools: [{google_search: {}}]` → ข้อความราคาที่เจอ + `groundingMetadata` (ลิงก์ที่ใช้ + `searchEntryPoint.renderedContent`) 2) รุ่นแชท + `generationConfig.responseJsonSchema` = `geminiReportSchema` (ช่องเดียวกับ `reportTool` ของ Claude, `reportProperties` ใช้ร่วมกัน) → `MarketReport`
  - แหล่งที่มาเก็บเฉพาะลิงก์ที่ Google ค้นเจอจริง (`foundSourcesOnly`, AI แต่งลิงก์ไม่ได้) / ขั้นที่ 2 ลองกับ Gemini จริงแล้วรับ schema ได้
  - **กติกาของ Google** (Gemini API Additional Terms): ผลที่ค้นผ่าน Google ต้องแสดง Search Suggestions (`search_suggestions` ใน `market_checks`, migration `000013`) คู่กับผลเสมอ ห้ามแก้ไข / เก็บไว้ได้เพื่อให้ผู้ใช้คนเดิมดูประวัติของตัวเอง — **หน้าเว็บยังไม่ได้ทำส่วนแสดงนี้ ต้องทำก่อนเปิด `GEMINI_WEB_SEARCH`**
- Gemini: เรียก `POST /models/{model}:generateContent` ด้วย Go standard library (ไม่เพิ่ม package), ส่งส่วนที่ AI ตอบกลับไปแบบเดิมทุกไบต์ (มี `thoughtSignature` ที่ Gemini ต้องได้คืน), คำตอบถูกบล็อก (SAFETY ฯลฯ) → 422 `ai_refused`

**key ของร้าน** (`PUT` / `DELETE /ai/key`):

- ต้องขึ้นต้นด้วย `sk-ant-` ไม่มีช่องว่าง (`validClaudeKey`) ไม่ใช่ → 400 `ai_key_format_invalid`
- ลอง key กับ Anthropic ก่อน (`Claude.CheckKey` → `Models.List` ไม่เสียเงิน) ไม่ผ่าน → 400 `ai_key_invalid` ไม่บันทึก — **key ที่เครดิตหมดยังผ่านขั้นนี้** (ขอรายชื่อรุ่นไม่ใช้เครดิต) จะรู้ตอนแชท / เช็คราคา → 503 `ai_credit_low`
- เก็บแบบเข้ารหัส AES-256-GCM (`utils.Encrypt` / `Decrypt`, กุญแจ = SHA-256 ของ `JWT_SECRET`) — API ไม่ส่ง key กลับเลย ส่งแค่ 4 ตัวท้าย (`key_hint`) / `GET /company` ไม่มี `ai_key`
- ถอดรหัสไม่ได้ (เช่น เปลี่ยน `JWT_SECRET`) → ลง log แล้วถือว่าไม่มี key (ร้านต้องใส่ใหม่) ไม่ทำให้ AI พัง

**Claude**:

- เปิด **fallback** ไว้: ถ้า Claude ปฏิเสธคำขอเพราะนโยบายความปลอดภัย จะให้รุ่นอื่นตอบแทนในคำขอเดียวกัน (`fallbacks: "default"`) / ยังปฏิเสธ = 422 `ai_refused`
- `Claude` (`Send`, `CheckKey`) และ `Gemini` (`Generate`) เป็น interface → test ใช้ตัวปลอม ไม่เรียก API จริง ไม่เสียเงิน
- คำสั่งของ AI (`ai/prompt.go`) เป็นภาษาไทย ไม่เปลี่ยนตามคำขอ → ใส่ `cache_control` ไว้ ประหยัดค่า AI (วันที่วันนี้แยกเป็นอีกก้อน)

**เช็คราคาตลาด** (`ai/market.go`): ส่งข้อมูลสินค้า + ใบต้นทุน + ช่องเพิ่มเติม + รูปแบบ VAT → Claude ค้นเว็บ (`web_search` ไม่เกิน 8 ครั้ง ตั้งที่อยู่เป็นไทย) → ส่งผลผ่านเครื่องมือ `save_market_report` (schema แบบ strict ได้ข้อมูลตรงรูปแบบเสมอ)

- Claude ค้นนานจนพักกลางทาง (`pause_turn`) → ส่งประวัติกลับไปให้ค้นต่อ / ตอบจบแต่ลืมส่งผล → เตือน 1 ครั้ง ยังไม่ส่ง = 502 `ai_no_result`
- `cleanReport` (`ai/logic.go`) ทำความสะอาดก่อนเก็บ: ตัวเลขติดลบ → 0, ต่ำสุด > สูงสุด → สลับ, ตัดลิงก์ที่ไม่ใช่ http(s), จำกัดรายการ / ความยาว
- เก็บทุกครั้งในตาราง `market_checks` (เก็บต้นทุน / ราคาของร้าน ณ วันที่เช็คไว้ด้วย) หน้าเว็บ / ภาพรวมใช้ครั้งล่าสุด

**แชท** (`ai/chat.go`): API ไม่เก็บแชท — Flutter ส่งประวัติทั้งหมดมาทุกครั้ง (เริ่มด้วย user สลับ assistant จบด้วย user ≤ 40 ข้อความ)

- AI ใช้เครื่องมือ (`ai/tools.go` → ทำงานใน `ai/toolbox.go`) วนได้ไม่เกิน 10 รอบต่อคำถาม
  - อ่านข้อมูล (ทำทันที): `get_shop_info`, `list_products`, `get_cost_sheet`, `get_market_check`, `get_dashboard`, `list_transactions` — ใช้ repository ของ feature อื่นผ่าน `ShopReader` (`ai/shop.go`) ไม่เขียน SQL ซ้ำ
  - เสนองาน (**ไม่ทำ** แค่ส่งกลับใน `actions`): `propose_create_product`, `propose_update_product`, `propose_stock_in`, `propose_stock_out`, `propose_transaction`, `propose_market_check` — ไม่เกิน 5 งานต่อคำตอบ
- งานที่เสนอตรวจด้วย `Validate()` เดียวกับ API เส้นจริง + เช็คว่ามีสินค้าจริง / ของพอ / หมวดถูก ผิด → ส่งกลับให้ AI แก้เอง (`tool_result` แบบ `is_error`) ไม่ใช่ error ของ request
- ผู้ใช้กดยืนยันใน Flutter → Flutter เรียก API เส้นเดิม (`POST /products`, `PUT /products/{id}`, `POST /stock/in|out`, `POST /transactions`, `POST /products/{id}/market`) — `update_product` ส่ง `data` เป็นสินค้าทั้งตัวหลังแก้ (PUT ได้เลย) + `changes` ไว้โชว์
- ลบสินค้า / ออก-ยกเลิกเอกสาร / แก้ใบต้นทุน / ตั้งค่าร้าน ทำผ่านแชทไม่ได้

### รายละเอียด transaction / invoice / dashboard (ทำแล้ว)

- `category` เป็นรหัสหมวด (Flutter แปลเป็นชื่อ) — API เช็คแค่ว่าห้ามว่าง รายรับจากใบกำกับใช้ `sales`
- รายการที่มี `invoice_id` (รายรับจากใบกำกับ) แก้ / ลบไม่ได้ → 409 `transaction_from_invoice` ต้องยกเลิกใบกำกับแทน
- ชนิดเอกสาร (`doc_type`) ดูจาก `company.vat_registered` ตอนออก: จด VAT → `tax_invoice` (`INV-2026-0001`), ไม่จด → `receipt` ใบเสร็จรับเงิน (`RC-2026-0001`) ไม่มี VAT
- ใบกำกับภาษี: ร้านต้องมีเลขผู้เสียภาษีก่อน (409 `company_tax_id_required`) และต้องมีชื่อลูกค้า (400 `invoice_customer_required`) / ใบเสร็จ: ไม่ต้องมีทั้งสองอย่าง (ไม่ใส่ชื่อ = ลูกค้าทั่วไป)
- `discount` = ส่วนลดทั้งบิล (0 ถึงยอดรวมสินค้า ไม่งั้น 400 `invoice_discount_invalid`) — `unit_price` รวม VAT หรือไม่ ดูจาก `company.prices_include_vat` (เก็บค่านี้ไว้ในใบด้วย)
- ตอบใบเต็ม (เหมือน `GET /invoices/{id}`): `items_total`, `discount`, `subtotal` (มูลค่าก่อน VAT), `vat`, `total`
- ยกเลิกเอกสาร: ได้ใบที่ `status = cancelled`, ยกเลิกซ้ำได้ 409 `invoice_already_cancelled`
- dashboard: รายได้ / รายจ่าย / กำไรสุทธิของเดือน (คิดแบบเดียวกับ `/transactions/summary`), จำนวน + ยอดใบกำกับที่ยังไม่ยกเลิก, จำนวนสินค้า, สินค้าใกล้หมด (ไม่เกิน 5 ตัว เหลือน้อยสุดก่อน)

### รายละเอียด company (ทำแล้ว)

**1 ระบบ = 1 ร้าน** — ร้านใหม่ติดตั้งระบบแยก (แบบ local) แล้วแก้ชื่อ / ที่อยู่ / โลโก้ผ่าน API หน้าเว็บ Flutter ใช้ตัวเดียวกันทุกร้าน ไม่ต้องแก้

- `GET /api/v1/company` → `{ "name", "address", "tax_id", "branch", "phone", "has_logo" }` (ยังไม่ตั้งค่า = ค่าว่างทุกช่อง)
- `PUT /api/v1/company` ส่งครบทุกช่อง (ยกเว้น `has_logo`) — `name` ห้ามว่าง, `tax_id` ว่างได้หรือเป็นตัวเลข 13 หลัก
- `GET /api/v1/company/logo` ตอบเป็นรูปตรง ๆ → `Image.network(...)` ได้เลย / 404 `logo_not_found` → แสดงโลโก้ CostBook แทน
- `PUT /api/v1/company/logo` → ได้ข้อมูลร้าน (`has_logo: true`) — เช็คชนิดไฟล์จากข้างในไฟล์ (`http.DetectContentType`) ไม่เชื่อนามสกุล, ไม่รับ SVG / GIF
- ดูข้อมูลร้าน / โลโก้ไม่ต้อง login เพื่อให้หน้า login แสดงชื่อ + โลโก้ร้านได้
- โลโก้ตอบ header `Cache-Control: no-cache` — Flutter ที่เพิ่งอัปโหลดโลโก้ใหม่ต้องโหลดรูปใหม่เอง (ล้าง cache ของ `Image.network`)

### รายละเอียด รูปแบบร้าน + ช่องข้อมูลเพิ่มเติม (ทำแล้ว)

ร้านแต่ละแบบคิดเงิน / ต้นทุน / ข้อมูลสินค้าไม่เหมือนกัน → ตั้งค่าได้เอง ไม่ต้องแก้โค้ด

| ตั้งค่า (ใน `company`) | ค่าเริ่มต้น | ผล |
|---|---|---|
| `vat_registered` | `true` | `true` ออกใบกำกับภาษี คิด VAT 7% / `false` ออกใบเสร็จรับเงิน ไม่คิด VAT ไม่ต้องมีเลขผู้เสียภาษี |
| `prices_include_vat` | `false` | `true` = ราคาขายที่ตั้งไว้รวม VAT แล้ว (ราคาหน้าร้าน) → ถอด VAT ออกจากยอด / `false` = บวก VAT เพิ่มจากราคา |
| `use_cost_page` | `true` | Flutter ซ่อนเมนูคำนวณต้นทุน (ร้านซื้อมาขายไปใช้ `unit_cost` ตอนรับเข้าแทน) — API ไม่ได้ใช้ค่านี้ |

- `PUT /company` **ไม่ส่ง** 3 ช่องนี้มา = ใช้ค่าเดิม (Go ใช้ `*bool`, `nil` → SQL `COALESCE($n, column)`)
- ช่องข้อมูลเพิ่มเติม (`/product-fields`): `{ "name", "type", "options", "show_in_list", "sort_order" }`
  - `type` = `text` / `number` / `date` / `select` (`select` ต้องมี `options` ไม่งั้น 400 `field_options_required`)
  - ค่าของแต่ละสินค้าเก็บใน `products.custom_fields` (JSONB) แบบ `{"<id ช่อง>": "ค่า"}` — **ไม่ต้อง ALTER TABLE** เพิ่ม column ทุกครั้งที่ร้านอยากได้ช่องใหม่
  - ลบช่อง → ลบ key นั้นออกจากทุกสินค้าใน transaction เดียว (SQL `custom_fields - $1::text`)
  - `show_in_list = true` → Flutter โชว์เป็น column ในตารางสินค้า

---

## 8. ตารางในฐานข้อมูล

| ตาราง | เก็บอะไร | column หลัก |
|---|---|---|
| `company` | ข้อมูลร้าน (มีแถวเดียว `id = 1`, migration `000002_create_company` สร้างแถวว่างไว้ให้) | name, address, tax_id, branch, phone, logo (`BYTEA` ว่างได้), logo_content_type, vat_registered, prices_include_vat, use_cost_page (`000007`), ai_key (key ของ Claude แบบเข้ารหัส ว่าง = ไม่มี `000009_ai_key`) |
| `products` | สินค้า (migration `000003_create_products`) | id, code (ห้ามซ้ำถ้าไม่ว่าง), name, category_id (→ `product_categories`, ลบหมวด = `NULL`, `000011` — เดิมเป็นข้อความ `category`), description, unit, cost_price, sell_price, stock (≥ 0), min_stock, batch_quantity (> 0), track_stock, for_sale (false = วัสดุ `000012`), custom_fields (`JSONB`), image (`BYTEA` ว่างได้) + image_content_type + image_version (`000010`) |
| `product_categories` | หมวดหมู่สินค้า (`000011_product_categories`, ชื่อห้ามซ้ำ `lower(btrim(name))`) | id, name |
| `product_fields` | ช่องข้อมูลเพิ่มเติมที่ร้านตั้งเอง (`000007_flexible_shop`) | id, name, type (text / number / date / select), options (`TEXT[]`), show_in_list, sort_order |
| `product_costs` | รายการต้นทุนของสินค้า (`000003`, ลบสินค้า → ลบด้วย) | id, product_id, line_no, type (material / labor / overhead), kind (fixed / percent), name, quantity, unit, unit_price, percent, amount, material_id (→ `products` วัสดุ ห้ามเป็นตัวเอง ลบวัสดุ = `NULL` `000012`) |
| `cost_presets` | รายการต้นทุนที่ใช้บ่อย (`000007`) | id, type, kind, name, unit, unit_price, percent |
| `invoices` | เอกสารขาย (`000004_create_invoices`) | id, doc_type (tax_invoice / receipt), invoice_no (ห้ามซ้ำ), issue_date, status (issued / cancelled), seller_* + customer_* ณ วันที่ออก, note, prices_include_vat, items_total, discount, subtotal, vat, total, cancelled_at |
| `invoice_items` | รายการในเอกสาร (`000004`, สินค้าที่อยู่ในนี้ลบไม่ได้) | id, invoice_id, line_no, product_id, name, unit, quantity, unit_price, amount, stock_cut (ตัด stock ไปไหม → ยกเลิกใบคืนเฉพาะที่ตัด), cost_price (ต้นทุน ณ วันที่ขาย `000008`) |
| `invoice_numbers` | ตัวรันเลขที่ (`000004`) | doc_type + year (คู่กันเป็น key), last_number |
| `market_checks` | ผลเช็คราคาตลาดด้วย AI (`000008_ai_market`, ลบสินค้า → ลบด้วย) | id, product_id, our_cost_price, our_sell_price, retail_low / high / typical, wholesale_low / high / note, market_cost, suggested_price, confidence (low / medium / high), summary, advice, cost_items (`JSONB`), sources (`JSONB`), created_at |
| `stock_movements` | ประวัติรับเข้า / เบิกออก (`000005_create_stock_movements`) | id, product_id, type (in / out), quantity, note, unit_cost, supplier, invoice_id, is_material (ตัด / คืนวัสดุอัตโนมัติ `000012`), created_at |
| `transactions` | รายรับ-รายจ่าย (`000006_create_transactions`) | id, type (income / expense), category, amount (> 0), date, note, invoice_id (ห้ามซ้ำ) |
| `users` | ผู้ใช้ที่ login ได้ (migration `000001_create_users`, role / ปิดบัญชี / รูปโปรไฟล์ `000014_user_roles`) | id, email (ห้ามซ้ำ), password_hash (bcrypt), first_name, last_name, role (`owner` / `manager` / `staff`), active, avatar (BYTEA), avatar_content_type, avatar_version |

- ทุกตารางมี `created_at` และ `updated_at`
- `line_no` = ลำดับบรรทัดตามที่ผู้ใช้กรอก — บันทึกหลายบรรทัดใน transaction เดียว `created_at` เท่ากันหมด เรียงด้วย `created_at` ไม่ได้
- migration รันเองด้วย `psql costbook -f migrations/<ไฟล์>.up.sql` ตามลำดับเลข (ยังไม่ได้ใช้ `golang-migrate`)
- `cost_price` = `utils.CostPerUnit(วัตถุดิบ, ค่าแรง, ค่าใช้จ่ายอื่น, batch_quantity)` คำนวณใหม่ทุกครั้งที่ `product_costs` เปลี่ยน หรือทุนของวัสดุที่ผูกไว้เปลี่ยน

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
| คำนวณ VAT | คิด **ทั้งใบ** หลังหักส่วนลด ไม่คิดทีละบรรทัด (`invoice/logic.go` → `documentAmounts`): ราคาไม่รวม VAT → `vat = utils.VAT(ยอดหลังลด)` / ราคารวม VAT แล้ว → `subtotal = utils.PriceBeforeVAT(ยอดหลังลด)`, `vat = ยอด − subtotal` / ใบเสร็จ → `vat = 0` |
| ตัด stock | เฉพาะสินค้าที่ `track_stock = true`: `UPDATE products SET stock = stock - $1 WHERE id = $2 AND stock >= $1` ถ้าไม่มีแถวไหนถูกแก้ = stock ไม่พอ → 409 `stock_not_enough` / สินค้าไม่นับสต็อกที่ใบต้นทุนผูกวัสดุ → ตัดวัสดุแทน (ไม่พอ 409 `material_not_enough`) / วัสดุ (`for_sale = false`) ขายไม่ได้ 409 `product_not_for_sale` |
| ตัดวัสดุตอนรับเข้า | รับสินค้าที่ผลิตเสร็จเข้าสต็อก → ตัดวัสดุตามใบต้นทุน (ไม่พอ = ไม่ให้รับเข้า เว้นแต่ส่ง `skip_materials`) — ดูหัวข้อ 7 "รายละเอียด วัสดุในสต็อก" |
| เลขที่เอกสาร | `INV-2026-0001` (ใบกำกับ) / `RC-2026-0001` (ใบเสร็จ) แยกเลขกัน รันใหม่ทุกปี ใช้ `INSERT ... ON CONFLICT (doc_type, year) DO UPDATE SET last_number = last_number + 1 RETURNING last_number` ใน transaction → เลขไม่ซ้ำแม้กดพร้อมกัน |
| แก้ / ลบใบกำกับ | **ห้าม** ถ้าผิดให้ยกเลิก (`status = cancelled`) แล้วออกใบใหม่ |
| ยกเลิกใบกำกับ | คืน stock + ลบรายรับที่ผูกกับใบนั้น (ทำใน transaction เดียว) |
| ข้อมูลในใบ | เก็บชื่อ ที่อยู่ เลขผู้เสียภาษี ของผู้ขาย / ผู้ซื้อ และชื่อสินค้า **ณ วันที่ออก** ไว้ในใบเลย แก้ข้อมูลบริษัทหรือสินค้าภายหลัง ใบเก่าต้องไม่เปลี่ยน — **ยกเว้นโลโก้** ไม่เก็บซ้ำในใบ (กฎหมายไม่บังคับ + DB จะบวม) พิมพ์ใบเก่าซ้ำจะได้โลโก้ปัจจุบัน |
| รายรับจากใบกำกับ | แก้ / ลบตรง ๆ ไม่ได้ ต้องยกเลิกใบกำกับ |
| ลบสินค้า | ถ้าเคยอยู่ในใบกำกับแล้ว ลบไม่ได้ → 409 `product_in_use` |
| เจ้าของร้าน | ต้องมีเจ้าของที่ใช้งานได้อย่างน้อย 1 คนเสมอ → ปลด / ปิดบัญชีเจ้าของคนสุดท้าย 409 `user_last_owner` / ไม่มีลบผู้ใช้ ใช้ปิดบัญชีแทน (ข้อมูลเก่าไม่หาย) |
| รายได้ / กำไร | **รายได้** = ยอดขายก่อน VAT หลังหักส่วนลด (เอกสารขายที่ไม่ยกเลิก) + รายรับที่บันทึกเอง (ไม่มี `invoice_id`) / **กำไรขั้นต้น** = ยอดขาย − ต้นทุนสินค้าที่ขาย (`invoice_items.cost_price` ณ วันที่ขาย) / **กำไรสุทธิ** = รายได้ − ต้นทุนสินค้าที่ขาย − รายจ่าย **ไม่รวมหมวดซื้อวัตถุดิบ** (`material` คิดอยู่ในต้นทุนสินค้าแล้ว ไม่หักซ้ำ — ผู้ใช้เลือก 2026-11-01) — คิดที่ `transaction.NewSummary` + SQL `transaction.MonthAmountsSQL` ตัวเดียว หน้ารายรับ-รายจ่าย / ภาพรวม / กราฟ 6 เดือนใช้ร่วมกัน / `income` / `expense` = เงินเข้า-ออกตามที่บันทึก (รายรับจากเอกสารขายรวม VAT) **ไม่ใช่กำไร** |

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
| `ANTHROPIC_API_KEY` | ไม่มี = ใช้ key ที่ร้านใส่ในหน้าตั้งค่า หรือ AI ฟรี | API key ของ Claude (สมัครที่ https://platform.claude.com) ใช้เช็คราคาตลาด + แชท — มีค่าใช้จ่ายตามการใช้งาน ห้าม commit |
| `GEMINI_API_KEY` | ไม่มี = ไม่มี AI ฟรี | API key ของ Gemini **ฟรี** (สร้างที่ https://aistudio.google.com/apikey) = AI เริ่มต้นเมื่อร้านไม่มี key ของ Claude — แชทได้ ค้นเว็บไม่ได้ ห้าม commit |
| `GEMINI_MODEL` | `gemini-3.8-flash` | รุ่นของ Gemini (เปลี่ยนเมื่อ Google ออกรุ่นใหม่ / รุ่นเดิมเลิกให้ใช้ฟรี) |
| `GEMINI_WEB_SEARCH` | ว่าง (ปิด) | `on` = ให้ Gemini ค้น Google ตอนเช็คราคาตลาด — ต้องเปิด Gemini แบบเสียเงินก่อน (แบบฟรีได้ 429) และหน้าเว็บต้องแสดง Search Suggestions ก่อน |
| `GEMINI_SEARCH_MODEL` | ว่าง (= `GEMINI_MODEL`) | รุ่นที่ใช้ค้น Google |
| `TAVILY_API_KEY` | ว่าง | ตัวค้นเว็บฟรี (tavily.com, 1,000 ครั้ง/เดือน) → Gemini ฟรีเช็คราคาตลาดได้ |
| `GEMINI_FALLBACK_MODELS` | ว่าง (= `gemini-3.5-flash,gemini-3.5-flash-lite`) | รุ่นสำรองเมื่อรุ่นหลักยุ่ง / โควตาเต็ม / ค้าง คั่นด้วย `,` |
| `JWT_SECRET` | **ต้องมี** | ข้อความลับไว้เซ็น token สร้างด้วย `openssl rand -hex 32` — แต่ละเครื่อง / server ใช้คนละค่า ถ้าเปลี่ยน token เก่าใช้ไม่ได้ทั้งหมด (ทุกคนต้อง login ใหม่) และ key AI ที่ร้านใส่ไว้ถอดรหัสไม่ได้ (ต้องใส่ใหม่) |
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
