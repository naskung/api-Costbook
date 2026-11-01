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
		{"แก้ข้อมูลร้าน ต้อง login ก่อน ได้ 401", http.MethodPut, "/api/v1/company", "", http.StatusUnauthorized, `{"error":"unauthorized"}`},
		{"อัปโหลดโลโก้ ต้อง login ก่อน ได้ 401", http.MethodPut, "/api/v1/company/logo", "", http.StatusUnauthorized, `{"error":"unauthorized"}`},
		{"ลบโลโก้ ต้อง login ก่อน ได้ 401", http.MethodDelete, "/api/v1/company/logo", "", http.StatusUnauthorized, `{"error":"unauthorized"}`},
		{"รายการสินค้า ต้อง login ก่อน ได้ 401", http.MethodGet, "/api/v1/products", "", http.StatusUnauthorized, `{"error":"unauthorized"}`},
		{"ช่องข้อมูลเพิ่มเติม ต้อง login ก่อน ได้ 401", http.MethodGet, "/api/v1/product-fields", "", http.StatusUnauthorized, `{"error":"unauthorized"}`},
		{"รายการต้นทุนที่ใช้บ่อย ต้อง login ก่อน ได้ 401", http.MethodGet, "/api/v1/cost-presets", "", http.StatusUnauthorized, `{"error":"unauthorized"}`},
		{"ใบต้นทุน ต้อง login ก่อน ได้ 401", http.MethodPut, "/api/v1/products/x/costs", "", http.StatusUnauthorized, `{"error":"unauthorized"}`},
		{"รับสินค้าเข้า ต้อง login ก่อน ได้ 401", http.MethodPost, "/api/v1/stock/in", "", http.StatusUnauthorized, `{"error":"unauthorized"}`},
		{"รายรับ-รายจ่าย ต้อง login ก่อน ได้ 401", http.MethodGet, "/api/v1/transactions", "", http.StatusUnauthorized, `{"error":"unauthorized"}`},
		{"ออกใบกำกับ ต้อง login ก่อน ได้ 401", http.MethodPost, "/api/v1/invoices", "", http.StatusUnauthorized, `{"error":"unauthorized"}`},
		{"ยกเลิกใบกำกับ ต้อง login ก่อน ได้ 401", http.MethodPost, "/api/v1/invoices/x/cancel", "", http.StatusUnauthorized, `{"error":"unauthorized"}`},
		{"ภาพรวม ต้อง login ก่อน ได้ 401", http.MethodGet, "/api/v1/dashboard", "", http.StatusUnauthorized, `{"error":"unauthorized"}`},
		{"แชท AI ต้อง login ก่อน ได้ 401", http.MethodPost, "/api/v1/ai/chat", "", http.StatusUnauthorized, `{"error":"unauthorized"}`},
		{"ใส่ key AI ต้อง login ก่อน ได้ 401", http.MethodPut, "/api/v1/ai/key", "", http.StatusUnauthorized, `{"error":"unauthorized"}`},
		{"เช็คราคาตลาด ต้อง login ก่อน ได้ 401", http.MethodPost, "/api/v1/products/x/market", "", http.StatusUnauthorized, `{"error":"unauthorized"}`},
		{"แก้โปรไฟล์ ต้อง login ก่อน ได้ 401", http.MethodPut, "/api/v1/profile", "", http.StatusUnauthorized, `{"error":"unauthorized"}`},
		{"จัดการผู้ใช้ ต้อง login ก่อน ได้ 401", http.MethodGet, "/api/v1/users", "", http.StatusUnauthorized, `{"error":"unauthorized"}`},
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
		if !strings.Contains(spec, "\n  "+openAPIPath(path)+":\n") {
			t.Errorf("docs/openapi.yaml ยังไม่มี %s %s", route.Method, route.Path)
		}
	}
}

// openAPIPath แปลง path แบบ Gin เป็นแบบ OpenAPI เช่น /products/:id → /products/{id}
func openAPIPath(ginPath string) string {
	parts := strings.Split(ginPath, "/")
	for i, part := range parts {
		name, isParam := strings.CutPrefix(part, ":")
		if isParam {
			parts[i] = "{" + name + "}"
		}
	}
	return strings.Join(parts, "/")
}

func TestOpenAPIPath(t *testing.T) {
	tests := []struct {
		ginPath string
		want    string
	}{
		{"/products", "/products"},
		{"/products/:id", "/products/{id}"},
		{"/invoices/:id/cancel", "/invoices/{id}/cancel"},
	}

	for _, tt := range tests {
		got := openAPIPath(tt.ginPath)
		if got != tt.want {
			t.Errorf("openAPIPath(%q) = %q, ต้องได้ %q", tt.ginPath, got, tt.want)
		}
	}
}
