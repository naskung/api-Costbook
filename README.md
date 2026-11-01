# api-Costbook

Backend (Go + Gin + PostgreSQL) ของ CostBook — คำนวณต้นทุนสินค้า, ใบกำกับภาษี, รายรับ-รายจ่าย, stock, ผู้ใช้ + ตำแหน่ง, AI
ฝั่งหน้าเว็บอยู่ที่ [../web-app-CostBook](../web-app-CostBook) — ทำอะไรไปแล้วบ้างดูใน [PROGRESS.md](PROGRESS.md)

## เปิดใช้ครั้งแรก (เครื่องใหม่ / ฐานข้อมูลใหม่)

1. มี Go + PostgreSQL (ดู [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) หัวข้อ 2)
2. สร้างฐานข้อมูล + ทุกตาราง (รันไฟล์ migration ตามลำดับเลข):
   ```bash
   createdb costbook
   for f in migrations/*.up.sql; do psql costbook -f "$f"; done
   ```
3. copy `.env.example` เป็น `.env` แล้วใส่ค่า — อย่างน้อย `DATABASE_URL` กับ `JWT_SECRET` (สร้างด้วย `openssl rand -hex 32`)
   ไฟล์ `.env` มี key ลับ **ห้าม commit / ห้ามส่งให้ใคร**
4. เพิ่มผู้ใช้คนแรก (เจ้าของร้าน): ดูตัวอย่างใน [sql/users.sql](sql/users.sql)
5. `make run` → เปิด http://localhost:8080/api/v1/health ได้ `{"status":"ok"}` = ใช้ได้

## คำสั่งที่ใช้บ่อย

```bash
make run      # เปิด API (อ่าน .env ให้เอง) — แก้โค้ด / ดึงโค้ดใหม่มา ต้องปิด (Ctrl+C) แล้วเปิดใหม่
make test     # รัน unit test ทั้งหมด
make check    # ตรวจก่อนส่งงาน: gofmt + go vet + test พร้อม % coverage
```

- เอกสาร API (Swagger) กดลองเรียกได้: http://localhost:8080/docs
- `go run ./cmd/api` ตรง ๆ จะ error เพราะไม่ได้อ่าน `.env` → ใช้ `make run`

## ฐานข้อมูล

```bash
# มี migration ไฟล์ใหม่ (เช่น 000014) → รันเฉพาะไฟล์ใหม่ แล้ว restart make run
psql costbook -f migrations/000014_user_roles.up.sql

# สำรองข้อมูล (ได้ไฟล์ .sql ไว้เก็บ)
pg_dump costbook > costbook-backup-$(date +%F).sql

# กู้ข้อมูลจากไฟล์สำรอง ลงฐานข้อมูลใหม่
createdb costbook_restore
psql costbook_restore < costbook-backup-2026-11-01.sql
```

คำสั่ง SQL ที่ใช้บ่อย (ดู / เพิ่ม / แก้ข้อมูล) อยู่ในโฟลเดอร์ [sql/](sql/)

## ให้เพื่อนลองเล่นผ่านเน็ต (ลิงก์ชั่วคราว)

เปิด `make run` ค้างไว้ แล้วในอีกหน้าต่าง:

```bash
cd ~/Documents/GitHub/costbook && caffeinate -i ./share-with-friend.sh
```

ได้ลิงก์ `https://xxx.trycloudflare.com` ส่งให้เพื่อน / เลิกเล่นกด Ctrl+C — รายละเอียดดู [../README.md](../README.md)

## เอกสาร

- โครงสร้างและกติกา: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- ความคืบหน้า (ทำถึงไหนแล้ว): [PROGRESS.md](PROGRESS.md)
- กฎสำหรับ Claude: [CLAUDE.md](CLAUDE.md)
- Skills: [simple-code](.claude/skills/simple-code/SKILL.md), [unit-test-check](.claude/skills/unit-test-check/SKILL.md)
