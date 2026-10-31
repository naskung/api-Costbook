package cost

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

// fakeRepository ฐานข้อมูลปลอม เก็บใบต้นทุนของสินค้า 1 ตัว ใช้แทน Postgres ตอน test
type fakeRepository struct {
	sheet          CostSheet
	savedCostPrice float64 // ต้นทุนต่อชิ้นที่ handler ส่งมาให้บันทึก
	err            error   // ใส่ค่านี้เมื่ออยากจำลองว่า DB พัง
}

func (f *fakeRepository) Get(ctx context.Context, productID string) (CostSheet, error) {
	if f.err != nil {
		return CostSheet{}, f.err
	}
	if productID != f.sheet.ProductID {
		return CostSheet{}, ErrProductNotFound
	}
	return f.sheet, nil
}

func (f *fakeRepository) Save(ctx context.Context, productID string, req CostSheetRequest, costPrice float64) (CostSheet, error) {
	sheet, err := f.Get(ctx, productID)
	if err != nil {
		return CostSheet{}, err
	}
	f.savedCostPrice = costPrice
	sheet.BatchQuantity = req.BatchQuantity
	sheet.CostPrice = costPrice
	sheet.SellPrice = req.SellPrice
	return sheet, nil
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
	return &fakeRepository{sheet: CostSheet{ProductID: coffeeID, ProductName: "กาแฟ", BatchQuantity: 1, Items: []CostItem{}}}
}

func TestHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		repo       *fakeRepository
		path       string
		wantStatus int
	}{
		{"เจอสินค้า ได้ 200", coffeeRepo(), "/api/v1/products/" + coffeeID + "/costs", http.StatusOK},
		{"ไม่มีสินค้านี้ ได้ 404", coffeeRepo(), "/api/v1/products/" + missingID + "/costs", http.StatusNotFound},
		{"id ไม่ใช่ UUID ได้ 404", coffeeRepo(), "/api/v1/products/abc/costs", http.StatusNotFound},
		{"DB พัง ได้ 500", &fakeRepository{err: errors.New("db down")}, "/api/v1/products/" + coffeeID + "/costs", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendRequest(newTestRouter(tt.repo), http.MethodGet, tt.path, "")
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, ต้องได้ %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestHandler_Get_RequiresLogin(t *testing.T) {
	router := newTestRouter(coffeeRepo())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/products/"+coffeeID+"/costs", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, ต้องได้ %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHandler_Save_CalculatesCostPrice(t *testing.T) {
	repo := coffeeRepo()
	body := `{"batch_quantity":100,"sell_price":15,"items":[
		{"type":"material","name":"เมล็ดกาแฟ","amount":500},
		{"type":"labor","name":"ค่าแรง","amount":300},
		{"type":"overhead","name":"ค่าไฟ","amount":200}]}`

	rec := sendRequest(newTestRouter(repo), http.MethodPut, "/api/v1/products/"+coffeeID+"/costs", body)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ต้องได้ %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	if repo.savedCostPrice != 10 {
		t.Errorf("ต้นทุนต่อชิ้นที่บันทึก = %v, ต้องได้ 10", repo.savedCostPrice)
	}
	var got CostSheet
	json.NewDecoder(rec.Body).Decode(&got)
	if got.CostPrice != 10 || got.SellPrice != 15 || got.BatchQuantity != 100 {
		t.Errorf("ได้ %+v, ต้องได้ cost_price 10, sell_price 15, batch_quantity 100", got)
	}
}

func TestHandler_Save_Errors(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{"JSON ผิดรูปแบบ ได้ 400", "/api/v1/products/" + coffeeID + "/costs", `{`, http.StatusBadRequest, `{"error":"invalid_json"}`},
		{"จำนวนผลิต 0 ได้ 400", "/api/v1/products/" + coffeeID + "/costs", `{"batch_quantity":0}`, http.StatusBadRequest, `{"error":"cost_batch_quantity_invalid"}`},
		{"ประเภทต้นทุนผิด ได้ 400", "/api/v1/products/" + coffeeID + "/costs", `{"batch_quantity":1,"items":[{"type":"x","name":"a","amount":1}]}`, http.StatusBadRequest, `{"error":"cost_type_invalid"}`},
		{"ไม่มีสินค้านี้ ได้ 404", "/api/v1/products/" + missingID + "/costs", `{"batch_quantity":1}`, http.StatusNotFound, `{"error":"product_not_found"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendRequest(newTestRouter(coffeeRepo()), http.MethodPut, tt.path, tt.body)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, ต้องได้ %d", rec.Code, tt.wantStatus)
			}
			if rec.Body.String() != tt.wantBody {
				t.Errorf("body = %s, ต้องได้ %s", rec.Body.String(), tt.wantBody)
			}
		})
	}
}
