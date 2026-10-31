package config

import (
	"errors"
	"os"
)

// Config ค่าที่ต่างกันแต่ละเครื่อง อ่านจาก environment variable (ตอนพัฒนาเก็บไว้ในไฟล์ .env)
// AnthropicAPIKey ไม่ใส่ก็ได้ = ปิดระบบ AI (เช็คราคาตลาด / แชท) ส่วนอื่นใช้ได้ปกติ
type Config struct {
	Port            string
	AllowedOrigin   string
	DatabaseURL     string
	JWTSecret       string
	AnthropicAPIKey string
}

// Load อ่านค่า config ทั้งหมด ถ้าขาดค่าที่จำเป็นจะคืน error
func Load() (Config, error) {
	cfg := Config{
		Port:          getEnv("PORT", "8080"),
		AllowedOrigin: getEnv("ALLOWED_ORIGIN", "*"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		// ชื่อเดียวกับที่ SDK ของ Claude ใช้
		AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is not set (run with: make run)")
	}
	if cfg.JWTSecret == "" {
		return Config{}, errors.New("JWT_SECRET is not set (run with: make run)")
	}
	return cfg, nil
}

// getEnv อ่าน environment variable ถ้าไม่มีให้ใช้ค่าเริ่มต้น
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
