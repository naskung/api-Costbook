package stock

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/middleware"
	"github.com/naskung/api-Costbook/internal/utils"
)

const (
	testSecret = "test-secret"
	coffeeID   = "11111111-1111-1111-1111-111111111111"
	missingID  = "99999999-9999-9999-9999-999999999999"
	serviceID  = "33333333-3333-3333-3333-333333333333"
)

// fakeRepository ฐานข้อมูลปลอม เก็บจำนวนคงเหลือของสินค้าใน map ใช้แทน Postgres ตอน test
type fakeRepository struct {
	stock      map[string]int // product_id → จำนวนคงเหลือ
	gotOnlyLow bool
	gotProduct string
	err        error // ใส่ค่านี้เมื่ออยากจำลองว่า DB พัง
}

func (f *fakeRepository) List(ctx context.Context, onlyLow bool) ([]StockItem, error) {
	f.gotOnlyLow = onlyLow
	return []StockItem{{ProductID: coffeeID, Name: "กาแฟ", Stock: f.stock[coffeeID]}}, f.err
}

func (f *fakeRepository) Movements(ctx context.Context, productID string) ([]Movement, error) {
	f.gotProduct = productID
	return []Movement{}, f.err
}

func (f *fakeRepository) AddMovement(ctx context.Context, movementType string, req MovementRequest) (Movement, error) {
	if f.err != nil {
		return Movement{}, f.err
	}
	current, found := f.stock[req.ProductID]
	if !found {
		return Movement{}, ErrProductNotFound
	}
	if req.ProductID == serviceID {
		return Movement{}, ErrNotTracked
	}
	change := req.Quantity
	if movementType == "out" {
		change = -req.Quantity
	}
	if current+change < 0 {
		return Movement{}, ErrNotEnough
	}
	f.stock[req.ProductID] = current + change
	return Movement{ID: "m-1", ProductID: req.ProductID, Type: movementType, Quantity: req.Quantity}, nil
}

func newTestRouter(repo Repository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler(repo).RegisterRoutes(r.Group("/api/v1"), middleware.Auth(testSecret))
	return r
}

func sendRequest(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	token, _ := utils.CreateToken("user-1", testSecret, time.Hour)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func coffeeRepo() *fakeRepository {
	return &fakeRepository{stock: map[string]int{coffeeID: 5, serviceID: 0}}
}

func TestHandler_List(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		wantOnlyLow bool
	}{
		{"ไม่ส่ง low ได้ทุกสินค้า", "/api/v1/stock", false},
		{"low=true ได้เฉพาะใกล้หมด", "/api/v1/stock?low=true", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := coffeeRepo()
			rec := sendRequest(newTestRouter(repo), http.MethodGet, tt.path, "")
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, ต้องได้ %d", rec.Code, http.StatusOK)
			}
			if repo.gotOnlyLow != tt.wantOnlyLow {
				t.Errorf("onlyLow = %v, ต้องได้ %v", repo.gotOnlyLow, tt.wantOnlyLow)
			}
		})
	}
}

func TestHandler_List_RequiresLogin(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter(coffeeRepo()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/stock", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, ต้องได้ %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHandler_List_DBError(t *testing.T) {
	rec := sendRequest(newTestRouter(&fakeRepository{err: errors.New("db down")}), http.MethodGet, "/api/v1/stock", "")

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, ต้องได้ %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHandler_Movements(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		wantStatus  int
		wantProduct string
	}{
		{"ไม่ส่ง product_id ได้ทุกสินค้า", "/api/v1/stock/movements", http.StatusOK, ""},
		{"ส่ง product_id ได้เฉพาะสินค้านั้น", "/api/v1/stock/movements?product_id=" + coffeeID, http.StatusOK, coffeeID},
		{"product_id ไม่ใช่ UUID ได้ 404", "/api/v1/stock/movements?product_id=abc", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := coffeeRepo()
			rec := sendRequest(newTestRouter(repo), http.MethodGet, tt.path, "")
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, ต้องได้ %d", rec.Code, tt.wantStatus)
			}
			if repo.gotProduct != tt.wantProduct {
				t.Errorf("product_id = %q, ต้องได้ %q", repo.gotProduct, tt.wantProduct)
			}
			if tt.wantStatus == http.StatusOK && rec.Body.String() != "[]" {
				t.Errorf("body = %s, ต้องได้ []", rec.Body.String())
			}
		})
	}
}

func TestHandler_StockInOut(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		body       string
		wantStatus int
		wantBody   string
		wantStock  int
	}{
		{"รับเข้า 10 ได้ 201 คงเหลือ 15", "/api/v1/stock/in", `{"product_id":"` + coffeeID + `","quantity":10}`, http.StatusCreated, "", 15},
		{"เบิกออก 5 ได้ 201 คงเหลือ 0", "/api/v1/stock/out", `{"product_id":"` + coffeeID + `","quantity":5}`, http.StatusCreated, "", 0},
		{"เบิกเกินที่มี ได้ 409", "/api/v1/stock/out", `{"product_id":"` + coffeeID + `","quantity":6}`, http.StatusConflict, `{"error":"stock_not_enough"}`, 5},
		{"ไม่มีสินค้านี้ ได้ 404", "/api/v1/stock/in", `{"product_id":"` + missingID + `","quantity":1}`, http.StatusNotFound, `{"error":"product_not_found"}`, 5},
		{"จำนวน 0 ได้ 400", "/api/v1/stock/in", `{"product_id":"` + coffeeID + `","quantity":0}`, http.StatusBadRequest, `{"error":"stock_quantity_invalid"}`, 5},
		{"JSON ผิดรูปแบบ ได้ 400", "/api/v1/stock/out", `{`, http.StatusBadRequest, `{"error":"invalid_json"}`, 5},
		{"สินค้าไม่นับสต็อก ได้ 409", "/api/v1/stock/in", `{"product_id":"` + serviceID + `","quantity":1}`, http.StatusConflict, `{"error":"stock_not_tracked"}`, 5},
		{"ราคาทุนติดลบ ได้ 400", "/api/v1/stock/in", `{"product_id":"` + coffeeID + `","quantity":1,"unit_cost":-1}`, http.StatusBadRequest, `{"error":"stock_unit_cost_invalid"}`, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := coffeeRepo()
			rec := sendRequest(newTestRouter(repo), http.MethodPost, tt.path, tt.body)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, ต้องได้ %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantBody != "" && rec.Body.String() != tt.wantBody {
				t.Errorf("body = %s, ต้องได้ %s", rec.Body.String(), tt.wantBody)
			}
			if repo.stock[coffeeID] != tt.wantStock {
				t.Errorf("คงเหลือ = %d, ต้องได้ %d", repo.stock[coffeeID], tt.wantStock)
			}
		})
	}
}

func TestHandler_StockIn_ReturnsMovement(t *testing.T) {
	rec := sendRequest(newTestRouter(coffeeRepo()), http.MethodPost, "/api/v1/stock/in", `{"product_id":"`+coffeeID+`","quantity":3}`)

	var got Movement
	json.NewDecoder(rec.Body).Decode(&got)
	if got.Type != "in" || got.Quantity != 3 {
		t.Errorf("ได้ %+v, ต้องได้ type in จำนวน 3", got)
	}
}
