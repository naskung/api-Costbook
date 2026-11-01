package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

// newAuthenticatedRouter route ที่ต้องผ่าน Authenticated (+ RequireRole ถ้าส่ง roles มา) แล้วตอบ role กลับมา
func newAuthenticatedRouter(lookup UserLookup, roles ...string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handlers := []gin.HandlerFunc{Authenticated("secret", lookup)}
	if len(roles) > 0 {
		handlers = append(handlers, RequireRole(roles...))
	}
	handlers = append(handlers, func(c *gin.Context) { c.String(http.StatusOK, c.GetString(RoleKey)) })
	r.GET("/private", handlers...)
	return r
}

func getPrivate(router *gin.Engine, withToken bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	if withToken {
		token, _ := utils.CreateToken("user-1", "secret", time.Hour)
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func lookupAs(role string, active bool, err error) UserLookup {
	return func(ctx context.Context, userID string) (string, bool, error) { return role, active, err }
}

func TestAuthenticated(t *testing.T) {
	tests := []struct {
		name       string
		lookup     UserLookup
		withToken  bool
		wantStatus int
		wantBody   string
	}{
		{"ผู้ใช้ปกติ ผ่าน + มี role ให้ handler", lookupAs("manager", true, nil), true, http.StatusOK, "manager"},
		{"บัญชีถูกปิด / ถูกลบ ได้ 401 ทันที", lookupAs("staff", false, nil), true, http.StatusUnauthorized, "unauthorized"},
		{"ไม่มี token ได้ 401 ไม่ต้องถาม DB", lookupAs("", false, errors.New("ห้ามถูกเรียก")), false, http.StatusUnauthorized, "unauthorized"},
		{"DB พัง ได้ 500", lookupAs("", false, errors.New("db down")), true, http.StatusInternalServerError, "server_error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := getPrivate(newAuthenticatedRouter(tt.lookup), tt.withToken)
			if rec.Code != tt.wantStatus || !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("ต้องได้ %d %s ได้ %d %s", tt.wantStatus, tt.wantBody, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	owner := getPrivate(newAuthenticatedRouter(lookupAs("owner", true, nil), "owner"), true)
	staff := getPrivate(newAuthenticatedRouter(lookupAs("staff", true, nil), "owner"), true)
	if owner.Code != http.StatusOK {
		t.Errorf("role ที่อนุญาต ต้องผ่าน ได้ %d", owner.Code)
	}
	if staff.Code != http.StatusForbidden || !strings.Contains(staff.Body.String(), "forbidden") {
		t.Errorf("role อื่น ต้องได้ 403 forbidden ได้ %d %s", staff.Code, staff.Body.String())
	}
}
