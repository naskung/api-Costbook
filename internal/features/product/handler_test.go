package product

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
)

// fakeRepository ฐานข้อมูลปลอม เก็บสินค้าใน slice ใช้แทน Postgres ตอน test
type fakeRepository struct {
	products []Product
	inUseID  string // id สินค้าที่จำลองว่าเคยอยู่ในใบกำกับ
	err      error  // ใส่ค่านี้เมื่ออยากจำลองว่า DB พัง
}

func (f *fakeRepository) List(ctx context.Context) ([]Product, error) {
	return f.products, f.err
}

func (f *fakeRepository) FindByID(ctx context.Context, id string) (Product, error) {
	if f.err != nil {
		return Product{}, f.err
	}
	for _, p := range f.products {
		if p.ID == id {
			return p, nil
		}
	}
	return Product{}, ErrNotFound
}

func (f *fakeRepository) Create(ctx context.Context, req ProductRequest) (Product, error) {
	if f.err != nil {
		return Product{}, f.err
	}
	p := Product{ID: "new-id", Name: req.Name, Unit: req.Unit, CostPrice: req.CostPrice, SellPrice: req.SellPrice, MinStock: req.MinStock, BatchQuantity: 1}
	f.products = append(f.products, p)
	return p, nil
}

func (f *fakeRepository) Update(ctx context.Context, id string, req ProductRequest) (Product, error) {
	p, err := f.FindByID(ctx, id)
	if err != nil {
		return Product{}, err
	}
	p.Name = req.Name
	p.SellPrice = req.SellPrice
	return p, nil
}

func (f *fakeRepository) Delete(ctx context.Context, id string) error {
	if id == f.inUseID {
		return ErrInUse
	}
	_, err := f.FindByID(ctx, id)
	return err
}

// newTestRouter สร้าง Gin router สำหรับ test: route เหมือนของจริง แต่ใช้ fakeRepository แทน DB
func newTestRouter(repo Repository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler(repo).RegisterRoutes(r.Group("/api/v1"), middleware.Auth(testSecret))
	return r
}

// sendRequest ยิง request พร้อม token (login แล้ว)
func sendRequest(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	token, _ := utils.CreateToken("user-1", testSecret, time.Hour)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func checkError(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantError string) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Errorf("status = %d, ต้องได้ %d", rec.Code, wantStatus)
	}
	wantBody := `{"error":"` + wantError + `"}`
	if rec.Body.String() != wantBody {
		t.Errorf("body = %s, ต้องได้ %s", rec.Body.String(), wantBody)
	}
}

func coffeeRepo() *fakeRepository {
	return &fakeRepository{products: []Product{{ID: coffeeID, Name: "กาแฟ", Unit: "แก้ว", SellPrice: 45, BatchQuantity: 1}}}
}

func TestHandler_RequiresLogin(t *testing.T) {
	router := newTestRouter(coffeeRepo())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	checkError(t, rec, http.StatusUnauthorized, "unauthorized")
}

func TestHandler_List(t *testing.T) {
	rec := sendRequest(newTestRouter(coffeeRepo()), http.MethodGet, "/api/v1/products", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ต้องได้ %d", rec.Code, http.StatusOK)
	}
	var got []Product
	json.NewDecoder(rec.Body).Decode(&got)
	if len(got) != 1 || got[0].Name != "กาแฟ" {
		t.Errorf("ได้ %+v, ต้องได้สินค้า 1 ตัวชื่อ กาแฟ", got)
	}
}

func TestHandler_List_Empty(t *testing.T) {
	rec := sendRequest(newTestRouter(&fakeRepository{products: []Product{}}), http.MethodGet, "/api/v1/products", "")

	if rec.Body.String() != "[]" {
		t.Errorf("body = %s, ต้องได้ [] (ไม่ใช่ null)", rec.Body.String())
	}
}

func TestHandler_List_DBError(t *testing.T) {
	rec := sendRequest(newTestRouter(&fakeRepository{err: errors.New("db down")}), http.MethodGet, "/api/v1/products", "")

	checkError(t, rec, http.StatusInternalServerError, "server_error")
}

func TestHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{"เจอสินค้า ได้ 200", "/api/v1/products/" + coffeeID, http.StatusOK},
		{"ไม่มีสินค้านี้ ได้ 404", "/api/v1/products/" + missingID, http.StatusNotFound},
		{"id ไม่ใช่ UUID ได้ 404", "/api/v1/products/abc", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendRequest(newTestRouter(coffeeRepo()), http.MethodGet, tt.path, "")
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, ต้องได้ %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{"ข้อมูลถูก ได้ 201", `{"name":"ชาไทย","unit":"แก้ว","cost_price":15,"sell_price":40,"min_stock":5}`, http.StatusCreated, ""},
		{"JSON ผิดรูปแบบ ได้ 400", `{"name":`, http.StatusBadRequest, `{"error":"invalid_json"}`},
		{"ชื่อว่าง ได้ 400", `{"name":"","sell_price":40}`, http.StatusBadRequest, `{"error":"product_name_required"}`},
		{"ราคาติดลบ ได้ 400", `{"name":"ชาไทย","sell_price":-1}`, http.StatusBadRequest, `{"error":"product_price_invalid"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendRequest(newTestRouter(coffeeRepo()), http.MethodPost, "/api/v1/products", tt.body)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, ต้องได้ %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantBody != "" && rec.Body.String() != tt.wantBody {
				t.Errorf("body = %s, ต้องได้ %s", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestHandler_Create_DBError(t *testing.T) {
	router := newTestRouter(&fakeRepository{err: errors.New("db down")})
	rec := sendRequest(router, http.MethodPost, "/api/v1/products", `{"name":"ชาไทย"}`)

	checkError(t, rec, http.StatusInternalServerError, "server_error")
}

func TestHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		body       string
		wantStatus int
	}{
		{"แก้สำเร็จ ได้ 200", "/api/v1/products/" + coffeeID, `{"name":"กาแฟเย็น","sell_price":50}`, http.StatusOK},
		{"ไม่มีสินค้านี้ ได้ 404", "/api/v1/products/" + missingID, `{"name":"กาแฟเย็น"}`, http.StatusNotFound},
		{"id ไม่ใช่ UUID ได้ 404", "/api/v1/products/abc", `{"name":"กาแฟเย็น"}`, http.StatusNotFound},
		{"ชื่อว่าง ได้ 400", "/api/v1/products/" + coffeeID, `{"name":" "}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendRequest(newTestRouter(coffeeRepo()), http.MethodPut, tt.path, tt.body)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, ต้องได้ %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestHandler_Delete(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		wantStatus int
		wantBody   string
	}{
		{"ลบสำเร็จ ได้ 204 ไม่มี body", coffeeID, http.StatusNoContent, ""},
		{"ไม่มีสินค้านี้ ได้ 404", missingID, http.StatusNotFound, `{"error":"product_not_found"}`},
		{"เคยอยู่ในใบกำกับ ได้ 409", "22222222-2222-2222-2222-222222222222", http.StatusConflict, `{"error":"product_in_use"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := coffeeRepo()
			repo.inUseID = "22222222-2222-2222-2222-222222222222"
			rec := sendRequest(newTestRouter(repo), http.MethodDelete, "/api/v1/products/"+tt.id, "")
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, ต้องได้ %d", rec.Code, tt.wantStatus)
			}
			if rec.Body.String() != tt.wantBody {
				t.Errorf("body = %s, ต้องได้ %s", rec.Body.String(), tt.wantBody)
			}
		})
	}
}
