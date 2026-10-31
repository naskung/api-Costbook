---
name: simple-code
description: กฎเขียนโค้ด Go แบบเรียบง่าย ไม่ซับซ้อน ของ CostBook API ใช้ทุกครั้งก่อนเขียน แก้ หรือรีวิวโค้ด Go ในโปรเจกต์นี้ ไม่ว่าจะเป็น handler, repository, model, middleware, router, utils หรือ helpers. Use whenever writing, editing, or reviewing any Go code in this repo.
---

# Simple Code (Go) — เขียนโค้ดให้ง่ายที่สุด

เป้าหมาย: คนที่ไม่เคยทำ backend อ่านแล้วเข้าใจได้ทันที

## หลักคิด

1. อ่านง่าย สำคัญกว่าสั้นหรือฉลาด
2. ทำเท่าที่ต้องใช้ตอนนี้ ไม่เขียนเผื่ออนาคต
3. ถ้ามีหลายวิธี เลือกวิธีที่ตรงไปตรงมาที่สุด
4. ทำตามโครงสร้างใน `docs/ARCHITECTURE.md` เสมอ ถ้าไม่แน่ใจว่าไฟล์ควรอยู่ไหน ให้เปิดดูหัวข้อ 4

## ทำ

- ใช้ **Gin** สำหรับ router, อ่าน / ตอบ JSON, middleware — นอกนั้นใช้ Go standard library ก่อนเสมอ (`testing`, `log/slog`, `os`)
- handler รับ `c *gin.Context` → อ่าน JSON ด้วย `c.ShouldBindJSON(&req)`, อ่าน path ด้วย `c.Param("id")`, ตอบด้วย `c.JSON(status, data)`
- ตอบ error ด้วย `helpers.WriteError(c, status, code)` / `helpers.ServerError(c, err)` เท่านั้น
- ส่ง `c.Request.Context()` ให้ repository (ไม่ส่ง `c` ทั้งก้อน)
- ฟังก์ชันละ 1 หน้าที่ ยาวไม่เกินประมาณ 30 บรรทัด
- เช็ค error ทันทีแบบ Go: `if err != nil { return ... }` แล้ว return เร็ว ไม่ซ้อน `if`
- ซ้อนกัน (nesting) ไม่เกิน 3 ชั้น
- เขียน SQL ตรง ๆ ใน repository อ่านแล้วรู้เลยว่าทำอะไร
- ใช้ `$1, $2` ส่งค่าเข้า SQL เสมอ
- ตั้งชื่อภาษาอังกฤษที่บอกความหมาย: `totalPrice` ไม่ใช่ `tp`, `products` ไม่ใช่ `list`
- โค้ดที่ใช้ตั้งแต่ 2 จุดขึ้นไป → แยกเป็นฟังก์ชันกลาง ห้าม copy-paste
  - ก่อนเขียนใหม่ ค้นใน `internal/utils`, `internal/helpers` ก่อนว่ามีอยู่แล้วไหม
  - ใช้ซ้ำแค่ในไฟล์ / package เดียว → ฟังก์ชันตัวเล็ก (`getEnv`) ใน package นั้น
  - ใช้ซ้ำหลาย package ไม่แตะ HTTP / DB → `internal/utils` (+ เขียน test)
  - ใช้ซ้ำหลาย package เกี่ยวกับ HTTP → `internal/helpers` (+ เขียน test)
- รัน `gofmt -w .` ก่อนส่งงานทุกครั้ง
- คอมเมนต์ภาษาไทยสั้น ๆ เหนือฟังก์ชันที่ใช้นอก package และจุดที่ไม่ชัด เช่น สูตรคำนวณ

## ห้าม

- ห้ามใช้ web framework ตัวอื่นนอกจาก Gin (Echo, Fiber, Chi) และ ORM (GORM)
- ห้ามใช้ `binding` tag / validator ของ Gin (`binding:"required"`) → เช็คใน `Validate()` แทน เพื่อคืน error code ของเราเอง
- ห้ามส่ง `*gin.Context` เข้า repository, logic หรือ utils
- ห้ามใช้ code generation (sqlc, wire, mockgen)
- ห้ามเพิ่มชั้นแบบ Clean Architecture (service, usecase, entity, dto แยก) — ใช้แค่ `handler → repository`
- ห้ามสร้าง interface ถ้าไม่ได้ใช้ใน test (ตอนนี้มีแค่ `Repository` ของแต่ละ feature)
- ห้ามใช้ generics ถ้ายังใช้ไม่ถึง 3 ที่
- ห้ามใช้ goroutine / channel เอง (Gin แยก request ให้อยู่แล้ว)
- ห้ามใช้ `panic` และ `init()` — คืน error แทน
- ห้ามใช้ตัวแปร global ที่เปลี่ยนค่าได้ (ใช้ได้แค่ `const` และ `var ErrXxx = errors.New(...)`)
- ห้ามต่อ string เป็น SQL (`"... WHERE id = '" + id + "'"`) → โดน SQL injection
- ห้ามส่ง error จริงของ server ให้ client → ใช้ `helpers.ServerError`
- ห้ามเขียนรหัสผ่าน / secret ลงในโค้ด → อ่านจาก environment variable
- ห้ามเพิ่ม package ใหม่โดยไม่ถามผู้ใช้ก่อน

## ตัวอย่าง

```go
// ❌ if ซ้อนกัน อ่านยาก
func (h *Handler) Get(c *gin.Context) {
	product, err := h.repo.FindByID(c.Request.Context(), c.Param("id"))
	if err == nil {
		c.JSON(http.StatusOK, product)
	} else {
		if errors.Is(err, ErrNotFound) {
			helpers.WriteError(c, http.StatusNotFound, "product_not_found")
		} else {
			helpers.ServerError(c, err)
		}
	}
}

// ✅ เช็ค error ก่อน แล้ว return เร็ว
func (h *Handler) Get(c *gin.Context) {
	product, err := h.repo.FindByID(c.Request.Context(), c.Param("id"))
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
```

```go
// ❌ copy สูตรเดียวกันไว้ 2 ที่
// invoice/logic.go
vat := math.Round(subtotal*7/100*100) / 100
// dashboard/logic.go
vat := math.Round(subtotal*7/100*100) / 100

// ✅ สูตรอยู่ที่เดียวใน internal/utils/tax.go ทั้ง 2 ที่เรียกใช้
vat := utils.VAT(subtotal)
```

```go
// ❌ ต่อ string เป็น SQL
r.db.Query(ctx, "SELECT * FROM products WHERE name = '"+name+"'")

// ✅ ส่งค่าผ่าน $1
r.db.Query(ctx, "SELECT id, name FROM products WHERE name = $1", name)
```

```
❌ ชั้นเยอะเกิน
Handler → ProductService → ProductUseCase → ProductRepositoryImpl → DB

✅ พอแล้ว
Handler → Repository → DB
```

## Checklist ก่อนส่งงาน

- [ ] คนที่ไม่เคยทำ backend อ่านแล้วเข้าใจไหม
- [ ] มีโค้ดที่ "เผื่อไว้" แต่ยังไม่ได้ใช้ไหม → ลบออก
- [ ] มีโค้ดซ้ำตั้งแต่ 2 จุดไหม → แยกเป็นฟังก์ชันกลาง
- [ ] ไฟล์อยู่ถูกโฟลเดอร์ตาม `docs/ARCHITECTURE.md`
- [ ] SQL ใช้ `$1, $2` ทุกที่ และ error ตอบเป็น `{"error": "code"}`
- [ ] `gofmt -l .` ไม่มีไฟล์ค้าง
- [ ] รัน skill `unit-test-check` แล้วผ่าน
- [ ] อัปเดต `PROGRESS.md` แล้ว
