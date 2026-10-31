package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect เปิดการเชื่อมต่อ PostgreSQL แล้วลอง ping ว่าต่อได้จริง
// pool = กลุ่มการเชื่อมต่อที่เปิดค้างไว้ ใช้ร่วมกันทุก request ไม่ต้องเปิดใหม่ทุกครั้ง
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	err = pool.Ping(ctx)
	if err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
