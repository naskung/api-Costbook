package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/middleware"
	"github.com/naskung/api-Costbook/internal/utils"
)

const testSecret = "test-secret"

// fakeRepository ฐานข้อมูลปลอม ใช้แทน Postgres ตอน test
type fakeRepository struct {
	gotMonthStart string
	err           error // ใส่ค่านี้เมื่ออยากจำลองว่า DB พัง
}

func (f *fakeRepository) Get(ctx context.Context, monthStart string) (Dashboard, error) {
	f.gotMonthStart = monthStart
	return Dashboard{Month: monthStart[:7], Income: 1000, Expense: 400, Profit: 600, LowStock: []LowStockItem{}}, f.err
}

func newTestRouter(repo Repository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler(repo).RegisterRoutes(r.Group("/api/v1"), middleware.Auth(testSecret))
	return r
}

func sendRequest(router *gin.Engine, path string, login bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if login {
		token, _ := utils.CreateToken("user-1", testSecret, time.Hour)
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestHandler_Get(t *testing.T) {
	repo := &fakeRepository{}
	rec := sendRequest(newTestRouter(repo), "/api/v1/dashboard?month=2026-10", true)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ต้องได้ %d", rec.Code, http.StatusOK)
	}
	if repo.gotMonthStart != "2026-10-01" {
		t.Errorf("monthStart = %q, ต้องได้ 2026-10-01", repo.gotMonthStart)
	}
	var got Dashboard
	json.NewDecoder(rec.Body).Decode(&got)
	if got.Month != "2026-10" || got.Profit != 600 {
		t.Errorf("ได้ %+v, ต้องได้เดือน 2026-10 กำไร 600", got)
	}
}

func TestHandler_Get_Errors(t *testing.T) {
	tests := []struct {
		name       string
		repo       *fakeRepository
		path       string
		login      bool
		wantStatus int
		wantBody   string
	}{
		{"ไม่ได้ login ได้ 401", &fakeRepository{}, "/api/v1/dashboard", false, http.StatusUnauthorized, `{"error":"unauthorized"}`},
		{"month ผิดรูปแบบ ได้ 400", &fakeRepository{}, "/api/v1/dashboard?month=abc", true, http.StatusBadRequest, `{"error":"month_invalid"}`},
		{"DB พัง ได้ 500", &fakeRepository{err: errors.New("db down")}, "/api/v1/dashboard", true, http.StatusInternalServerError, `{"error":"server_error"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendRequest(newTestRouter(tt.repo), tt.path, tt.login)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, ต้องได้ %d", rec.Code, tt.wantStatus)
			}
			if rec.Body.String() != tt.wantBody {
				t.Errorf("body = %s, ต้องได้ %s", rec.Body.String(), tt.wantBody)
			}
		})
	}
}
