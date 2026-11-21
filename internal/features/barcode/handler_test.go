package barcode

import (
	"context"
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

// fakeFinder ตัวหาปลอม ใช้แทน Open Food Facts ตอน test
type fakeFinder struct {
	product Product
	err     error
	gotCode string
}

func (f *fakeFinder) Find(ctx context.Context, code string) (Product, error) {
	f.gotCode = code
	return f.product, f.err
}

func sendRequest(finder Finder, path string, withToken bool) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(finder).RegisterRoutes(router.Group("/api/v1"), middleware.Auth(testSecret))

	req := httptest.NewRequest(http.MethodGet, path, nil)
	if withToken {
		token, _ := utils.CreateToken("user-1", testSecret, time.Hour)
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestHandler_RequiresLogin(t *testing.T) {
	rec := sendRequest(&fakeFinder{}, "/api/v1/barcodes/8850124003850", false)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, ต้องได้ %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHandler_Get(t *testing.T) {
	found := Product{Code: "8850124003850", Name: "Nestle Pure Life", Brand: "Nestlé", Quantity: "600 mL"}
	tests := []struct {
		name       string
		code       string
		finder     *fakeFinder
		wantStatus int
		wantBody   string
	}{
		{"เจอสินค้า ได้ 200 + ชื่อ / ยี่ห้อ / ขนาด", "8850124003850", &fakeFinder{product: found}, http.StatusOK,
			`{"code":"8850124003850","name":"Nestle Pure Life","brand":"Nestlé","quantity":"600 mL"}`},
		{"ไม่เจอ ได้ 404", "8850000000001", &fakeFinder{err: ErrNotFound}, http.StatusNotFound, `{"error":"barcode_not_found"}`},
		{"Open Food Facts ล่ม ได้ 502", "8850124003850", &fakeFinder{err: errors.New("timeout")}, http.StatusBadGateway, `{"error":"barcode_lookup_failed"}`},
		{"รหัสที่ร้านตั้งเอง (ไม่ใช่บาร์โค้ด) ได้ 400 ไม่ไปถาม", "CF001", &fakeFinder{}, http.StatusBadRequest, `{"error":"barcode_invalid"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendRequest(tt.finder, "/api/v1/barcodes/"+tt.code, true)
			if rec.Code != tt.wantStatus || rec.Body.String() != tt.wantBody {
				t.Errorf("ได้ %d %s, ต้องได้ %d %s", rec.Code, rec.Body.String(), tt.wantStatus, tt.wantBody)
			}
		})
	}
}

func TestHandler_Get_InvalidCodeDoesNotCallFinder(t *testing.T) {
	finder := &fakeFinder{}
	sendRequest(finder, "/api/v1/barcodes/abc", true)
	if finder.gotCode != "" {
		t.Errorf("เลขไม่ใช่บาร์โค้ด ต้องไม่ไปถามฐานข้อมูลภายนอก แต่ถามด้วย %q", finder.gotCode)
	}
}
