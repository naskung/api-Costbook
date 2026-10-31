package docs

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r)

	tests := []struct {
		name            string
		path            string
		wantContentType string
		wantText        string
	}{
		{"หน้า Swagger ได้ HTML", "/docs", "text/html; charset=utf-8", "swagger-ui"},
		{"ไฟล์เอกสาร ได้ YAML", "/docs/openapi.yaml", "application/yaml; charset=utf-8", "openapi: 3.0.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, ต้องได้ %d", rec.Code, http.StatusOK)
			}
			if rec.Header().Get("Content-Type") != tt.wantContentType {
				t.Errorf("Content-Type = %q, ต้องได้ %q", rec.Header().Get("Content-Type"), tt.wantContentType)
			}
			if !strings.Contains(rec.Body.String(), tt.wantText) {
				t.Errorf("body ไม่มีคำว่า %q", tt.wantText)
			}
		})
	}
}
