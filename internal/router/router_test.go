package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/config"
)

// test เหล่านี้ไม่แตะ DB จึงส่ง db เป็น nil ได้
func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return New(config.Config{AllowedOrigin: "*", JWTSecret: "secret"}, nil)
}

func TestRouter(t *testing.T) {
	router := newTestRouter()

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{"health ได้ 200", http.MethodGet, "/api/v1/health", "", http.StatusOK, `{"status":"ok"}`},
		{"path ที่ไม่มี ได้ 404", http.MethodGet, "/api/v1/nothing", "", http.StatusNotFound, `{"error":"not_found"}`},
		{"login มีอยู่จริง (JSON ผิด ได้ 400)", http.MethodPost, "/api/v1/auth/login", "{", http.StatusBadRequest, `{"error":"invalid_json"}`},
		{"me ต้อง login ก่อน ได้ 401", http.MethodGet, "/api/v1/auth/me", "", http.StatusUnauthorized, `{"error":"unauthorized"}`},
		{"browser ถามสิทธิ์ (OPTIONS) ได้ 204", http.MethodOptions, "/api/v1/auth/login", "", http.StatusNoContent, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, ต้องได้ %d", rec.Code, tt.wantStatus)
			}
			if rec.Body.String() != tt.wantBody {
				t.Errorf("body = %s, ต้องได้ %s", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestRouter_Docs(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /docs status = %d, ต้องได้ %d", rec.Code, http.StatusOK)
	}
}

// ทุก route ใน /api/v1 ต้องมีในเอกสาร docs/openapi.yaml
// (เพิ่ม endpoint ใหม่แล้วลืมเขียนเอกสาร test นี้จะไม่ผ่าน)
func TestRouter_AllRoutesInOpenAPI(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/docs/openapi.yaml", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /docs/openapi.yaml status = %d, ต้องได้ %d", rec.Code, http.StatusOK)
	}
	spec := rec.Body.String()

	for _, route := range router.Routes() {
		path, found := strings.CutPrefix(route.Path, "/api/v1")
		if !found {
			continue
		}
		if !strings.Contains(spec, "\n  "+path+":\n") {
			t.Errorf("docs/openapi.yaml ยังไม่มี %s %s", route.Method, route.Path)
		}
	}
}
