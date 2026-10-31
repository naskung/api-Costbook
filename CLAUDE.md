# CostBook API (Go)

Backend ของ CostBook: คำนวณต้นทุนสินค้า, ใบกำกับภาษี, รายรับ-รายจ่าย, stock
ฝั่งเว็บ (Flutter) อยู่ที่ `../web-app-CostBook` — ผู้ใช้ยังไม่เคยทำ backend มาก่อน อธิบายด้วยภาษาง่าย

## อ่านก่อนเริ่มงานทุกครั้ง

1. `docs/ARCHITECTURE.md` — โครงสร้าง กติกา API และกฎทางธุรกิจ
2. `PROGRESS.md` — ทำอะไรไปแล้ว และงานถัดไปคืออะไร

## กฎหลัก

- **ทำ API ทีละส่วน เฉพาะที่ผู้ใช้สั่งเท่านั้น** ห้ามเริ่มเขียนโค้ด / เพิ่ม endpoint / เพิ่ม Phase ถัดไปเองล่วงหน้า ทำเสร็จแล้วหยุดรอคำสั่ง
- เขียนโค้ดแบบ **simple** เสมอ → ใช้ skill `simple-code`
- เขียน / แก้โค้ดเสร็จ → ใช้ skill `unit-test-check` ก่อนบอกว่างานเสร็จ
- ทำงานเสร็จ → อัปเดต `PROGRESS.md` (ติ๊ก checklist + เพิ่มบรรทัดในบันทึกการทำงาน)
- โครงสร้าง: `handler → repository → PostgreSQL`, คำนวณอยู่ใน `utils` / `logic.go`, ไม่มีชั้น service
- web framework ใช้ **Gin** ส่วนอื่นใช้ Go standard library ก่อนเสมอ จะเพิ่ม package ใหม่ → ถามผู้ใช้ก่อน
- JSON `snake_case` ตรงกับ model ฝั่ง Flutter, error ตอบเป็น `{"error": "code"}`
- เพิ่ม / แก้ endpoint → แก้ `docs/openapi.yaml` ด้วย (ดูได้ที่ `http://localhost:8080/docs`)
- แก้ API ที่กระทบ Flutter → แจ้งผู้ใช้ และบันทึกใน `PROGRESS.md` ของทั้ง 2 repo
- ตอบผู้ใช้เป็นภาษาไทย

## คำสั่ง

```bash
make run          # เปิด API (โหลดค่าจาก .env ให้) — go run ./cmd/api ตรง ๆ จะ error เพราะไม่มี DATABASE_URL / JWT_SECRET
gofmt -l .
go vet ./...
go test ./...
```
