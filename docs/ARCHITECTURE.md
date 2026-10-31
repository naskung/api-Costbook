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
| HTTP server | โปรแกรมที่เปิดรอรับ request ที่ port หนึ่ง | `net/http` (มากับ Go) | 1 |
| Router | จับคู่ "method + URL" กับฟังก์ชันที่จะทำงาน | `http.ServeMux` (มากับ Go) | 1 |
| JSON | รูปแบบข้อมูลที่ส่งไปกลับกับ Flutter | `encoding/json` (มากับ Go) | 1 |
| CORS | อนุญาตให้เว็บที่อยู่คนละ port / โดเมนเรียก API ได้ ถ้าไม่มี browser จะบล็อก | `middleware/cors.go` | 1 |
| Error format | ตอบ error แบบเดียวกันทุกที่ Flutter อ่านง่าย | `{"error": "code"}` | 1 |
| Config | ค่าที่ต่างกันแต่ละเครื่อง (port, รหัส DB) อ่านจาก environment variable ไม่เขียนลงโค้ด | `config/config.go` | 1 |
| Unit test | test อัตโนมัติ เช็คว่าโค้ดยังถูกอยู่ | `testing` + `httptest` (มากับ Go) | 1 และทุก Phase |
| Database | ที่เก็บข้อมูลถาวร | PostgreSQL + `pgx` | 2 |
| Migration | ไฟล์ SQL สร้าง / แก้ตาราง เรียงตามลำดับ ทุกเครื่องได้ตารางเหมือนกัน | `golang-migrate` | 2 |
| Handler | ฟังก์ชันที่รับ request แล้วตอบกลับ (1 endpoint = 1 ฟังก์ชัน) | `features/*/handler.go` | 3 |
| Validation | เช็คข้อมูลที่ส่งมาก่อนบันทึก เช่น ชื่อห้ามว่าง ราคาห้ามติดลบ | `Validate()` ใน model | 3 |
| DB Transaction | หลายคำสั่ง SQL ต้องสำเร็จพร้อมกัน หรือยกเลิกทั้งหมด (เช่น ออกใบกำกับ + ตัด stock) | `pgx` Begin / Commit | 3 |
| Auth | login และเช็คว่าใครเรียก API | JWT + bcrypt | 4 |
| Logging | พิมพ์ว่ามี request อะไรเข้ามา ไว้หาจุดผิด | `log/slog` (มากับ Go) | 4 |
| Deploy | เอา API ขึ้น server จริง + HTTPS + สำรองข้อมูล | ยังไม่เลือก | 5 |

---

## 2. เครื่องมือที่ต้องติดตั้ง

| เครื่องมือ | สถานะ | วิธีติดตั้ง |
|---|---|---|
| Go | มีแล้ว (1.26) | — |
| PostgreSQL | ยังไม่มี (Phase 2) | `brew install postgresql@17` แล้ว `brew services start postgresql@17` |
| golang-migrate | ยังไม่มี (Phase 2) | `brew install golang-migrate` |
| ตัวยิง API ทดสอบ | มี `curl` แล้ว | หรือใช้ Postman / Bruno ถ้าอยากได้หน้าจอ |
| ตัวดูข้อมูลใน DB | ไม่บังคับ | TablePlus หรือ DBeaver |
| VS Code extension | แนะนำ | "Go" (ของทีม Go) — จัดรูปแบบโค้ดและรัน test ได้ในตัว |

---

## 3. Package ที่ใช้

| Package | ใช้ทำอะไร | Phase |
|---|---|---|
| Go standard library | server, router, JSON, test, log | 1 |
| `github.com/jackc/pgx/v5` | ต่อ PostgreSQL | 2 |
| `golang-migrate` (CLI ไม่ใช่ package) | รันไฟล์ migration | 2 |
| `github.com/golang-jwt/jwt/v5` | token สำหรับ login | 4 |
| `golang.org/x/crypto/bcrypt` | เข้ารหัสรหัสผ่าน | 4 |

**ไม่ใช้:** Gin / Echo / Fiber, GORM (ORM), sqlc, wire, testify
เหตุผล: Go 1.22 ขึ้นไป `net/http` ทำ router ได้เองแล้ว ใช้ของมาตรฐานจะเรียนรู้น้อยกว่า และอ่าน SQL ตรง ๆ เข้าใจง่ายกว่า ORM

---

## 4. โครงสร้างโฟลเดอร์

```
api-Costbook/
├── cmd/api/main.go           จุดเริ่มโปรแกรม: อ่าน config → สร้าง router → เปิด server
├── internal/                 (Go ไม่ให้โปรเจกต์อื่น import โค้ดในนี้)
│   ├── config/               อ่านค่าจาก environment variable
│   ├── database/             เชื่อมต่อ PostgreSQL (Phase 2)
│   ├── router/               รวม route ทั้งหมดไว้ที่เดียว
│   ├── middleware/           ทำงานก่อนทุก request: cors, logger, auth
│   ├── helpers/              ช่วยงาน HTTP: WriteJSON, WriteError, ServerError
│   ├── utils/                ฟังก์ชันคำนวณล้วน ๆ: money, tax, cost
│   └── features/
│       ├── company/
│       ├── product/
│       │   ├── model.go           struct + Validate()
│       │   ├── model_test.go
│       │   ├── repository.go      คุยกับ DB (interface + Postgres)
│       │   ├── handler.go         รับ request / ตอบ JSON
│       │   ├── handler_test.go    test ด้วย fakeRepository (ไม่ต้องมี DB)
│       │   ├── logic.go           (ถ้ามี) คำนวณเฉพาะ feature นี้
│       │   └── logic_test.go
│       ├── stock/
│       ├── transaction/
│       ├── invoice/
│       └── dashboard/
├── migrations/               ไฟล์ SQL เช่น 000001_create_products.up.sql / .down.sql
├── docs/ARCHITECTURE.md      ไฟล์นี้
├── PROGRESS.md               ทำถึงไหนแล้ว
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
Request → middleware (cors, auth) → router → handler ──▶ repository ──▶ PostgreSQL
                                              │
                                              └─▶ Validate() / logic / utils (คำนวณ)
```

| ชั้น | ทำ | ห้ามทำ |
|---|---|---|
| handler | อ่าน JSON, เรียก `Validate()`, เรียก repository, ตอบ JSON + status | เขียน SQL |
| model | struct + `Validate()` | เรียก DB |
| logic / utils | คำนวณ | แตะ HTTP หรือ DB |
| repository | SQL และ transaction | รู้จัก `http.Request` |

ไม่มีชั้น service แยก — ถ้า feature ไหนต้องทำหลายขั้นใน DB (เช่น ออกใบกำกับ) ให้ทำใน repository ภายใน transaction เดียว

---

## 6. กติกาของ API

- ทุก path ขึ้นต้นด้วย `/api/v1`
- ชื่อ field ใน JSON เป็น `snake_case` และ **ต้องตรงกับ model ฝั่ง Flutter** เช่น `cost_price`, `min_stock`
- `id` เป็น UUID (ส่งเป็น string)
- เงิน: DB เก็บเป็น `NUMERIC(12,2)` ส่วนใน Go ใช้ `float64` และปัดด้วย `utils.Round2` ทุกครั้งที่คำนวณเสร็จ
- วันที่ `"2026-10-31"`, วันเวลา `"2026-10-31T10:00:00Z"`
- รายการว่างต้องตอบ `[]` ไม่ใช่ `null` → สร้าง slice ด้วย `products := []Product{}`
- สำเร็จ: ตอบข้อมูลตรง ๆ (object หรือ array) ไม่ห่อด้วย `{"data": ...}`
- error: `{"error": "product_name_required"}` — เป็น **code ภาษาอังกฤษ** ฝั่ง Flutter แปลเป็นข้อความด้วย `'error.<code>'.tr`

| Method | ใช้ทำ | status เมื่อสำเร็จ |
|---|---|---|
| GET | อ่าน | 200 |
| POST | สร้าง | 201 |
| PUT | แก้ไข | 200 |
| DELETE | ลบ | 204 (ไม่มี body) |

| status error | เมื่อไหร่ | ตัวอย่าง code |
|---|---|---|
| 400 | ข้อมูลที่ส่งมาผิด | `invalid_json`, `product_name_required` |
| 401 | ยังไม่ login | `unauthorized` |
| 404 | ไม่เจอข้อมูล | `product_not_found` |
| 409 | ขัดกับข้อมูลที่มีอยู่ | `stock_not_enough`, `product_in_use` |
| 500 | server พัง | `server_error` (ใช้ `helpers.ServerError` → error จริงลง log ไม่ส่งให้ client) |

---

## 7. Endpoints

| Feature | Method + Path | ทำอะไร |
|---|---|---|
| health | `GET /api/v1/health` | เช็คว่า API ทำงาน |
| company | `GET /api/v1/company` | ข้อมูลบริษัท (ผู้ขาย) |
| | `PUT /api/v1/company` | แก้ข้อมูลบริษัท |
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
| auth (Phase 4) | `POST /api/v1/auth/login` | login ได้ token |
| | `GET /api/v1/auth/me` | ข้อมูลผู้ใช้ที่ login อยู่ |

---

## 8. ตารางในฐานข้อมูล

| ตาราง | เก็บอะไร | column หลัก |
|---|---|---|
| `company` | ข้อมูลบริษัท (มีแถวเดียว) | name, address, tax_id, branch, phone |
| `products` | สินค้า | id, name, unit, cost_price, sell_price, stock, min_stock, batch_quantity |
| `product_costs` | รายการต้นทุนของสินค้า | id, product_id, type (material / labor / overhead), name, amount |
| `stock_movements` | ประวัติรับเข้า / เบิกออก | id, product_id, type (in / out), quantity, note, invoice_id, created_at |
| `transactions` | รายรับ-รายจ่าย | id, type (income / expense), category, amount, date, note, invoice_id |
| `invoices` | ใบกำกับภาษี | id, invoice_no, issue_date, status, ข้อมูลผู้ขาย + ผู้ซื้อ, subtotal, vat, total |
| `invoice_items` | รายการในใบกำกับ | id, invoice_id, product_id, name, quantity, unit_price, amount |
| `invoice_numbers` | ตัวรันเลขที่ใบกำกับ | year, last_number |
| `users` (Phase 4) | ผู้ใช้ | id, email, password_hash |

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

// List GET /api/v1/products
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	products, err := h.repo.List(r.Context())
	if err != nil {
		helpers.ServerError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, products)
}

// Get GET /api/v1/products/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	product, err := h.repo.FindByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		helpers.WriteError(w, http.StatusNotFound, "product_not_found")
		return
	}
	if err != nil {
		helpers.ServerError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, product)
}

// Create POST /api/v1/products
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req ProductRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	code := req.Validate()
	if code != "" {
		helpers.WriteError(w, http.StatusBadRequest, code)
		return
	}

	product, err := h.repo.Create(r.Context(), req)
	if err != nil {
		helpers.ServerError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusCreated, product)
}
```

```go
// internal/router/router.go — ต่อ route ของ feature
productHandler := product.NewHandler(product.NewPostgresRepository(db))
mux.HandleFunc("GET /api/v1/products", productHandler.List)
mux.HandleFunc("GET /api/v1/products/{id}", productHandler.Get)
mux.HandleFunc("POST /api/v1/products", productHandler.Create)
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
| ข้อมูลในใบ | เก็บชื่อ ที่อยู่ เลขผู้เสียภาษี ของผู้ขาย / ผู้ซื้อ และชื่อสินค้า **ณ วันที่ออก** ไว้ในใบเลย แก้ข้อมูลบริษัทหรือสินค้าภายหลัง ใบเก่าต้องไม่เปลี่ยน |
| รายรับจากใบกำกับ | แก้ / ลบตรง ๆ ไม่ได้ ต้องยกเลิกใบกำกับ |
| ลบสินค้า | ถ้าเคยอยู่ในใบกำกับแล้ว ลบไม่ได้ → 409 `product_in_use` |

---

## 11. Unit Test

- `go test ./...` ต้องรันได้ **โดยไม่ต้องมี DB** — handler test ใช้ `fakeRepository` แทน
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
| `DATABASE_URL` | (Phase 2) | เช่น `postgres://postgres:postgres@localhost:5432/costbook?sslmode=disable` |

ตัวอย่าง: `PORT=9000 go run ./cmd/api`
รหัสผ่าน / secret ห้ามเขียนในโค้ด และห้าม commit ไฟล์ `.env` (อยู่ใน `.gitignore` แล้ว)

---

## 14. คำสั่งที่ใช้บ่อย

```bash
go run ./cmd/api                          # เปิด API ที่ http://localhost:8080
curl localhost:8080/api/v1/health         # ลองเรียก → {"status":"ok"}
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
- PDF ใบกำกับ สร้างฝั่ง Flutter จากข้อมูล `GET /api/v1/invoices/{id}`
- แก้ API ที่กระทบ Flutter (เปลี่ยนชื่อ field / path) → แก้ทั้ง 2 repo และบันทึกใน `PROGRESS.md` ของทั้ง 2 ฝั่ง
