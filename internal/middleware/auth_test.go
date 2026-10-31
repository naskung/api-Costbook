package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/utils"
)

// newAuthTestRouter router ที่มี route เดียว ต้องผ่าน Auth ก่อน แล้วตอบ id ผู้ใช้กลับมา
func newAuthTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/private", Auth("secret"), func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString(UserIDKey))
	})
	return r
}

func TestAuth_ValidToken(t *testing.T) {
	token, _ := utils.CreateToken("user-1", "secret", time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	newAuthTestRouter().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ต้องได้ %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "user-1" {
		t.Errorf("id ผู้ใช้ = %q, ต้องได้ %q", rec.Body.String(), "user-1")
	}
}

func TestAuth_Unauthorized(t *testing.T) {
	validToken, _ := utils.CreateToken("user-1", "secret", time.Hour)
	expiredToken, _ := utils.CreateToken("user-1", "secret", -time.Minute)

	tests := []struct {
		name   string
		header string
	}{
		{"ไม่มี header ได้ 401", ""},
		{"ไม่ขึ้นต้นด้วย Bearer ได้ 401", validToken},
		{"token มั่ว ได้ 401", "Bearer not-a-token"},
		{"token หมดอายุ ได้ 401", "Bearer " + expiredToken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/private", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			newAuthTestRouter().ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, ต้องได้ %d", rec.Code, http.StatusUnauthorized)
			}
			if rec.Body.String() != `{"error":"unauthorized"}` {
				t.Errorf("body = %s, ต้องได้ %s", rec.Body.String(), `{"error":"unauthorized"}`)
			}
		})
	}
}
