# api-Costbook

Backend (Go + Gin + PostgreSQL) ของ CostBook — คำนวณต้นทุนสินค้า, ใบกำกับภาษี, รายรับ-รายจ่าย, stock

ตอนนี้มี: login (`POST /api/v1/auth/login`, `GET /api/v1/auth/me`) และ `GET /api/v1/health` (ดูสถานะทั้งหมดใน [PROGRESS.md](PROGRESS.md))

## วิธีรัน

1. มี PostgreSQL และ database `costbook` (ดู [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) หัวข้อ 2)
2. สร้างตาราง: `psql costbook -f migrations/000001_create_users.up.sql`
3. copy `.env.example` เป็น `.env` แล้วใส่ `JWT_SECRET` (สร้างด้วย `openssl rand -hex 32`)
4. `make run` → เปิด http://localhost:8080/api/v1/health
5. ดูเอกสาร API (Swagger) และลองเรียก: http://localhost:8080/docs

## เอกสาร

- โครงสร้างและกติกา: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- ความคืบหน้า (ทำถึงไหนแล้ว): [PROGRESS.md](PROGRESS.md)
- กฎสำหรับ Claude: [CLAUDE.md](CLAUDE.md)
- Skills: [simple-code](.claude/skills/simple-code/SKILL.md), [unit-test-check](.claude/skills/unit-test-check/SKILL.md)
