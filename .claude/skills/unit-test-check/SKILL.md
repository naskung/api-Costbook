---
name: unit-test-check
description: ตรวจสอบ unit test ของ CostBook API (Go) — รัน gofmt, go vet, go test, หาไฟล์ที่ยังไม่มี test, เขียน test แบบง่ายให้ utils / helpers / middleware / model / handler แล้วรายงานผล ใช้ทุกครั้งหลังเขียนหรือแก้โค้ด Go เสร็จ ก่อนบอกว่างานเสร็จ หรือเมื่อผู้ใช้ขอ "ตรวจ test", "รัน test", "เช็ค unit test api". Use after any Go code change and whenever asked to check or run tests.
---

# Unit Test Check (Go)

ถ้ายังไม่มีไฟล์ `.go` ในโปรเจกต์ → แจ้งผู้ใช้ว่ายังไม่มีโค้ดให้ตรวจ แล้วหยุด (ห้ามเขียนโค้ดเพิ่มเอง)

## ขั้นตอน (ทำตามลำดับ)

1. **จัดรูปแบบโค้ด** — ต้องไม่มีไฟล์ค้าง (ถ้ามี รัน `gofmt -w .`)
   ```bash
   gofmt -l .
   ```
2. **ตรวจโค้ด** — ต้องไม่มีข้อความแจ้งเตือน
   ```bash
   go vet ./...
   ```
3. **รัน test** — ต้องผ่านทั้งหมด
   ```bash
   go test -cover ./...
   ```
4. **หาไฟล์ที่ยังไม่มี test** — `xxx.go` ต้องมี `xxx_test.go` อยู่ข้างกัน (ดูตารางว่าไฟล์ไหนบังคับ)
   ```bash
   for f in $(find internal -name '*.go' ! -name '*_test.go' ! -name 'repository.go'); do
     t="${f%.go}_test.go"
     [ -f "$t" ] || echo "ยังไม่มี test: $f"
   done
   ```
5. **เขียน test ที่ขาด** ตามแม่แบบด้านล่าง แล้วกลับไปทำข้อ 3
6. **รายงานผล** ตามรูปแบบท้ายไฟล์ แล้วอัปเดต `PROGRESS.md`

## อะไรต้องมี test

| ไฟล์ | บังคับ? | ทดสอบอะไร |
|---|---|---|
| `internal/utils/*.go` | บังคับ | ทุกฟังก์ชัน: ค่าปกติ + ค่าขอบ (0, ติดลบ, หารไม่ลงตัว) |
| `internal/helpers/*.go` | บังคับ | status, header, body ที่ตอบออกไป |
| `internal/middleware/*.go` | บังคับ | ส่งต่อ / ไม่ส่งต่อ request, header ที่ใส่ |
| `features/*/model.go` | บังคับ ถ้ามี `Validate()` | ข้อมูลถูก → `""`, ข้อมูลผิดแต่ละแบบ → code ที่ถูก |
| `features/*/logic.go` | บังคับ | ทุกฟังก์ชันคำนวณ |
| `features/*/handler.go` | บังคับ | สำเร็จ, JSON ผิด (400), validate ไม่ผ่าน (400), ไม่เจอ (404), DB พัง (500) |
| `internal/router/router.go` | บังคับ | route มีอยู่จริง และ path ที่ไม่มีได้ 404 `{"error":"not_found"}` |
| `features/*/repository.go` | ไม่ใช่ unit test | ต้องมี DB จริง → integration test (Phase 4, ไม่บังคับ) |
| `cmd/api/main.go` | ไม่ต้อง | — |

## กฎการเขียน test

- ใช้แค่ `testing` + `net/http/httptest` (มากับ Go) + Gin ไม่ใช้ testify / gomock
- test ที่ใช้ Gin เรียก `gin.SetMode(gin.TestMode)` ก่อนสร้าง router (ปิดข้อความ debug ของ Gin)
- handler test ยิงผ่าน router ที่ต่อด้วย `RegisterRoutes` ของ feature นั้น ไม่เรียก handler ตรง ๆ
- `go test ./...` ต้องรันได้ **โดยไม่ต้องมี DB และไม่ต้องต่อเน็ต** → handler ใช้ `fakeRepository`
- ฟังก์ชันที่มีหลายกรณี เขียนแบบ table-driven + `t.Run`
- ชื่อกรณีเป็นภาษาไทย บอกผลที่ต้องได้ เช่น `"100 บาท ได้ VAT 7 บาท"`
- ข้อความ error บอกทั้งค่าที่ได้และค่าที่ต้องได้: `t.Errorf("VAT(%v) = %v, ต้องได้ %v", ...)`
- ใช้ `t.Fatalf` เมื่อทำต่อไม่ได้แล้ว (เช่น status ผิด ไม่ต้องเช็ค body ต่อ) นอกนั้นใช้ `t.Errorf`
- ใช้ `t.Setenv` เปลี่ยน environment variable ใน test (คืนค่าเดิมให้เอง)
- **ห้าม** ลบ test, ใส่ `t.Skip`, หรือแก้ค่าที่ต้องได้เพื่อให้ผ่าน — ให้แก้โค้ดให้ถูก
  ถ้า test เขียนผิดจริง แก้ได้ แต่ต้องบอกผู้ใช้ว่าแก้เพราะอะไร

## แม่แบบ

### Utils (table-driven)

```go
// internal/utils/tax_test.go
package utils

import "testing"

func TestVAT(t *testing.T) {
	tests := []struct {
		name  string
		price float64
		want  float64
	}{
		{"100 บาท ได้ VAT 7 บาท", 100, 7},
		{"0 บาท ได้ VAT 0 บาท", 0, 0},
		{"1,234.56 บาท ได้ VAT 86.42 บาท", 1234.56, 86.42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VAT(tt.price)
			if got != tt.want {
				t.Errorf("VAT(%v) = %v, ต้องได้ %v", tt.price, got, tt.want)
			}
		})
	}
}
```

### Model — `Validate()`

```go
// internal/features/product/model_test.go
package product

import "testing"

func TestProductRequest_Validate(t *testing.T) {
	tests := []struct {
		name string
		req  ProductRequest
		want string
	}{
		{"ข้อมูลครบ ผ่าน", ProductRequest{Name: "กาแฟ", Unit: "แก้ว", CostPrice: 20, SellPrice: 45}, ""},
		{"ชื่อว่าง ไม่ผ่าน", ProductRequest{Name: "", CostPrice: 20, SellPrice: 45}, "product_name_required"},
		{"ชื่อมีแต่ช่องว่าง ไม่ผ่าน", ProductRequest{Name: "   ", CostPrice: 20, SellPrice: 45}, "product_name_required"},
		{"ราคาติดลบ ไม่ผ่าน", ProductRequest{Name: "กาแฟ", CostPrice: -1, SellPrice: 45}, "product_price_invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.req.Validate()
			if got != tt.want {
				t.Errorf("Validate() = %q, ต้องได้ %q", got, tt.want)
			}
		})
	}
}
```

### Handler — ใช้ `fakeRepository` แทน DB

```go
// internal/features/product/handler_test.go
package product

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// fakeRepository ฐานข้อมูลปลอม เก็บข้อมูลใน slice ใช้แทน Postgres ตอน test
type fakeRepository struct {
	products []Product
	err      error // ใส่ค่านี้เมื่ออยากจำลองว่า DB พัง
}

func (f *fakeRepository) List(ctx context.Context) ([]Product, error) {
	return f.products, f.err
}

func (f *fakeRepository) FindByID(ctx context.Context, id string) (Product, error) {
	if f.err != nil {
		return Product{}, f.err
	}
	for _, p := range f.products {
		if p.ID == id {
			return p, nil
		}
	}
	return Product{}, ErrNotFound
}

func (f *fakeRepository) Create(ctx context.Context, req ProductRequest) (Product, error) {
	if f.err != nil {
		return Product{}, f.err
	}
	p := Product{ID: "new-id", Name: req.Name, Unit: req.Unit, CostPrice: req.CostPrice, SellPrice: req.SellPrice}
	f.products = append(f.products, p)
	return p, nil
}

// newTestRouter สร้าง Gin router สำหรับ test: route เหมือนของจริง แต่ใช้ fakeRepository แทน DB
func newTestRouter(repo Repository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler(repo).RegisterRoutes(r.Group("/api/v1"))
	return r
}

func TestHandler_List(t *testing.T) {
	repo := &fakeRepository{products: []Product{{ID: "1", Name: "กาแฟ"}}}
	router := newTestRouter(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ต้องได้ %d", rec.Code, http.StatusOK)
	}
	var got []Product
	json.NewDecoder(rec.Body).Decode(&got)
	if len(got) != 1 || got[0].Name != "กาแฟ" {
		t.Errorf("ได้ %+v, ต้องได้สินค้า 1 ตัวชื่อ กาแฟ", got)
	}
}

func TestHandler_List_DBError(t *testing.T) {
	router := newTestRouter(&fakeRepository{err: errors.New("db down")})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, ต้องได้ %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHandler_Get_NotFound(t *testing.T) {
	router := newTestRouter(&fakeRepository{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products/999", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req) // ผ่าน router → c.Param("id") ได้ "999" เอง

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, ต้องได้ %d", rec.Code, http.StatusNotFound)
	}
}

func TestHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"ข้อมูลถูก ได้ 201", `{"name":"กาแฟ","unit":"แก้ว","cost_price":20,"sell_price":45}`, http.StatusCreated},
		{"JSON ผิดรูปแบบ ได้ 400", `{"name":`, http.StatusBadRequest},
		{"ชื่อว่าง ได้ 400", `{"name":"","cost_price":20,"sell_price":45}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newTestRouter(&fakeRepository{})

			req := httptest.NewRequest(http.MethodPost, "/api/v1/products", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, ต้องได้ %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}
```

### Middleware

```go
// internal/middleware/cors_test.go
func TestCORS_OptionsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS("http://localhost:3000"))
	r.OPTIONS("/api/v1/products", func(c *gin.Context) {
		t.Error("OPTIONS ไม่ควรถูกส่งต่อไปที่ handler")
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/products", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, ต้องได้ %d", rec.Code, http.StatusNoContent)
	}
}
```

## เมื่อ test ไม่ผ่าน

1. อ่านบรรทัด `--- FAIL:` จะบอกชื่อ test และข้อความ "ได้ ... ต้องได้ ..."
2. ดูว่าโค้ดผิด หรือ test ผิด
3. แก้ทีละจุด แล้วรันเฉพาะ test นั้น เช่น `go test ./internal/utils -run TestVAT -v`
4. ผ่านแล้ว รัน `go test ./...` ทั้งหมดอีกรอบ

## รูปแบบรายงานผล

```
## ผลตรวจ Unit Test (API)

| ขั้นตอน | ผล |
|---|---|
| gofmt -l . | ผ่าน (ไม่มีไฟล์ค้าง) |
| go vet ./... | ผ่าน |
| go test ./... | ผ่านทุก package |
| coverage | utils 100%, helpers 100%, product 85% |
| ไฟล์ที่ยังไม่มี test | ไม่มี |

test ที่เพิ่มใหม่: internal/features/product/handler_test.go (6 tests)
```

ถ้ามี test ไม่ผ่าน ให้ระบุชื่อ test, ไฟล์ และสาเหตุทีละข้อ
