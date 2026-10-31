# โหลดค่าจากไฟล์ .env (ถ้ามี) แล้วส่งให้ทุกคำสั่งข้างล่าง
-include .env
export

.PHONY: run test check

# เปิด API
run:
	go run ./cmd/api

# รัน unit test ทั้งหมด
test:
	go test ./...

# ตรวจโค้ดทั้งหมดก่อนส่งงาน
check:
	gofmt -l .
	go vet ./...
	go test -cover ./...
