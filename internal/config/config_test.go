package config

import "testing"

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("ALLOWED_ORIGIN", "")
	t.Setenv("DATABASE_URL", "postgres://localhost/costbook")
	t.Setenv("JWT_SECRET", "secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, ต้องไม่มี error", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, ต้องได้ %q", cfg.Port, "8080")
	}
	if cfg.AllowedOrigin != "*" {
		t.Errorf("AllowedOrigin = %q, ต้องได้ %q", cfg.AllowedOrigin, "*")
	}
	if cfg.DatabaseURL != "postgres://localhost/costbook" {
		t.Errorf("DatabaseURL = %q, ต้องได้ %q", cfg.DatabaseURL, "postgres://localhost/costbook")
	}
	if cfg.JWTSecret != "secret" {
		t.Errorf("JWTSecret = %q, ต้องได้ %q", cfg.JWTSecret, "secret")
	}
}

func TestLoad_CustomValues(t *testing.T) {
	t.Setenv("PORT", "9000")
	t.Setenv("ALLOWED_ORIGIN", "https://costbook.example.com")
	t.Setenv("DATABASE_URL", "postgres://localhost/costbook")
	t.Setenv("JWT_SECRET", "secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, ต้องไม่มี error", err)
	}
	if cfg.Port != "9000" {
		t.Errorf("Port = %q, ต้องได้ %q", cfg.Port, "9000")
	}
	if cfg.AllowedOrigin != "https://costbook.example.com" {
		t.Errorf("AllowedOrigin = %q, ต้องได้ %q", cfg.AllowedOrigin, "https://costbook.example.com")
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	tests := []struct {
		name        string
		databaseURL string
		jwtSecret   string
	}{
		{"ไม่มี DATABASE_URL ได้ error", "", "secret"},
		{"ไม่มี JWT_SECRET ได้ error", "postgres://localhost/costbook", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", tt.databaseURL)
			t.Setenv("JWT_SECRET", tt.jwtSecret)

			_, err := Load()
			if err == nil {
				t.Error("Load() error = nil, ต้องได้ error")
			}
		})
	}
}

func TestLoad_AIValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/costbook")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("GEMINI_WEB_SEARCH", "on")
	t.Setenv("TAVILY_API_KEY", "tvly-test")
	t.Setenv("GEMINI_FALLBACK_MODELS", " gemini-a , ,gemini-b")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.GeminiWebSearch || cfg.TavilyAPIKey != "tvly-test" {
		t.Errorf("อ่านค่า AI ไม่ครบ ได้ %+v", cfg)
	}
	if len(cfg.GeminiFallbackModels) != 2 || cfg.GeminiFallbackModels[0] != "gemini-a" || cfg.GeminiFallbackModels[1] != "gemini-b" {
		t.Errorf("รุ่นสำรองต้องแยกด้วย , ตัดช่องว่าง ข้ามค่าว่าง ได้ %q", cfg.GeminiFallbackModels)
	}
}

func TestLoad_AIDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/costbook")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("GEMINI_WEB_SEARCH", "")
	t.Setenv("GEMINI_FALLBACK_MODELS", "")

	cfg, _ := Load()
	if cfg.GeminiWebSearch || cfg.GeminiFallbackModels != nil {
		t.Errorf("ไม่ตั้งค่า = ปิดค้น Google + ใช้รุ่นสำรองค่าเริ่มต้น ได้ %+v", cfg)
	}
}
