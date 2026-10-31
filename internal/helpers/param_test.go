package helpers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUUIDParam(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantOK     bool
		wantStatus int
		wantBody   string
	}{
		{"id เป็น UUID ผ่าน", "/products/a00d265d-5466-4cba-aba7-005953ae30fc", true, http.StatusOK, "a00d265d-5466-4cba-aba7-005953ae30fc"},
		{"id ไม่ใช่ UUID ได้ 404 พร้อม code", "/products/123", false, http.StatusNotFound, `{"error":"product_not_found"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.GET("/products/:id", func(c *gin.Context) {
				id, ok := UUIDParam(c, "id", "product_not_found")
				if ok != tt.wantOK {
					t.Errorf("ok = %v, ต้องได้ %v", ok, tt.wantOK)
				}
				if ok {
					c.String(http.StatusOK, id)
				}
			})

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, ต้องได้ %d", rec.Code, tt.wantStatus)
			}
			if rec.Body.String() != tt.wantBody {
				t.Errorf("body = %s, ต้องได้ %s", rec.Body.String(), tt.wantBody)
			}
		})
	}
}
