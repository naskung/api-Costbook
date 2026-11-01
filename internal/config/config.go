package config

import (
	"errors"
	"os"
	"strings"
)

// Config ค่าที่ต่างกันแต่ละเครื่อง อ่านจาก environment variable (ตอนพัฒนาเก็บไว้ในไฟล์ .env)
// AnthropicAPIKey / GeminiAPIKey ไม่ใส่ก็ได้ — ร้านใส่ key ของ Claude เองที่หน้าตั้งค่าได้, Gemini = AI ฟรีตอนร้านไม่มี key
type Config struct {
	Port            string
	AllowedOrigin   string
	DatabaseURL     string
	JWTSecret       string
	AnthropicAPIKey string
	GeminiAPIKey    string
	GeminiModel     string
	// GeminiWebSearch = เปิดให้ Gemini ค้น Google ตอนเช็คราคาตลาด (GEMINI_WEB_SEARCH=on)
	// แบบฟรีค้น Google ไม่ได้ (โควตา 0) → เปิดเมื่อเปิดใช้ Gemini แบบเสียเงินแล้วเท่านั้น
	GeminiWebSearch bool
	// GeminiSearchModel รุ่นที่ใช้ค้น Google (ไม่ใส่ = รุ่นเดียวกับ GEMINI_MODEL)
	GeminiSearchModel string
	// TavilyAPIKey ตัวค้นเว็บฟรี (1,000 ครั้ง/เดือน) ให้ Gemini ฟรีเช็คราคาตลาดได้ — ไม่ใส่ = ไม่ใช้
	TavilyAPIKey string
	// GeminiFallbackModels รุ่นสำรองเมื่อรุ่นหลักยุ่ง / โควตาเต็ม (GEMINI_FALLBACK_MODELS คั่นด้วย , ไม่ใส่ = ค่าเริ่มต้นของ ai)
	GeminiFallbackModels []string
}

// Load อ่านค่า config ทั้งหมด ถ้าขาดค่าที่จำเป็นจะคืน error
func Load() (Config, error) {
	cfg := Config{
		Port:          getEnv("PORT", "8080"),
		AllowedOrigin: getEnv("ALLOWED_ORIGIN", "*"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		// ชื่อเดียวกับที่ SDK ของ Claude ใช้
		AnthropicAPIKey:      os.Getenv("ANTHROPIC_API_KEY"),
		GeminiAPIKey:         os.Getenv("GEMINI_API_KEY"),
		GeminiModel:          os.Getenv("GEMINI_MODEL"),
		GeminiWebSearch:      os.Getenv("GEMINI_WEB_SEARCH") == "on",
		GeminiSearchModel:    os.Getenv("GEMINI_SEARCH_MODEL"),
		TavilyAPIKey:         os.Getenv("TAVILY_API_KEY"),
		GeminiFallbackModels: splitList(os.Getenv("GEMINI_FALLBACK_MODELS")),
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

// splitList "a, b,,c" → [a b c] (ว่าง = nil)
func splitList(value string) []string {
	var items []string
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}
