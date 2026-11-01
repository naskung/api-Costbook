package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORS_OptionsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS("http://localhost:3000"))
	r.OPTIONS("/api/v1/products", func(c *gin.Context) {
		t.Error("OPTIONS ไม่ควรถูกส่งต่อไปที่ handler")
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/products", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, ต้องได้ %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Max-Age"); got != "600" {
		t.Errorf("browser ต้องจำคำตอบ OPTIONS ไว้ 10 นาที (ไม่ถามซ้ำทุก request) ได้ %q", got)
	}
}

func TestCORS_NormalRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS("http://localhost:3000"))
	r.GET("/api/v1/products", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, ต้องได้ %d (request ต้องถูกส่งต่อไปที่ handler)", rec.Code, http.StatusOK)
	}

	headers := map[string]string{
		"Access-Control-Allow-Origin":  "http://localhost:3000",
		"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization",
	}
	for name, want := range headers {
		got := rec.Header().Get(name)
		if got != want {
			t.Errorf("header %s = %q, ต้องได้ %q", name, got, want)
		}
	}
}
