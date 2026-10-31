package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/naskung/api-Costbook/internal/config"
	"github.com/naskung/api-Costbook/internal/database"
	"github.com/naskung/api-Costbook/internal/router"
)

func main() {
	err := run()
	if err != nil {
		slog.Error("เปิด server ไม่ได้", "error", err)
		os.Exit(1)
	}
}

// run อ่าน config → ต่อ DB → สร้าง router → เปิด server
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	db, err := database.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	r := router.New(cfg, db)
	slog.Info("API พร้อมใช้งาน", "url", "http://localhost:"+cfg.Port+"/api/v1/health")
	return r.Run(":" + cfg.Port)
}
