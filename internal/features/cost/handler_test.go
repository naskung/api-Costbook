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
	plaID      = "22222222-2222-2222-2222-222222222222"
	missingID  = "99999999-9999-9999-9999-999999999999"
)

// fakeRepository ฐานข้อมูลปลอม เก็บใบต้นทุนของสินค้า 1 ตัว + รายการที่ใช้บ่อย ใช้แทน Postgres ตอน test
type fakeRepository struct {
	sheet          CostSheet
	presets        []CostPreset
	savedAmounts   []float64 // จำนวนเงินแต่ละบรรทัดที่ handler ส่งมาให้บันทึก
	savedCostPrice float64   // ต้นทุนต่อชิ้นที่ handler ส่งมาให้บันทึก
	err            error     // ใส่ค่านี้เมื่ออยากจำลองว่า DB พัง
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

func (f *fakeRepository) Save(ctx context.Context, productID string, req CostSheetRequest, amounts []float64, costPrice float64) (CostSheet, error) {
	sheet, err := f.Get(ctx, productID)
	if err != nil {
		return CostSheet{}, err
	}
	f.savedAmounts = amounts
	f.savedCostPrice = costPrice
	sheet.BatchQuantity = req.BatchQuantity
	sheet.CostPrice = costPrice
	sheet.SellPrice = req.SellPrice
	return sheet, nil
}

func (f *fakeRepository) ListPresets(ctx context.Context) ([]CostPreset, error) {
	return f.presets, f.err
}

func (f *fakeRepository) CreatePreset(ctx context.Context, req PresetRequest) (CostPreset, error) {
	if f.err != nil {
		return CostPreset{}, f.err
	}
	return CostPreset{ID: "new-id", Type: req.Type, Kind: req.Kind, Name: req.Name, Unit: req.Unit, UnitPrice: req.UnitPrice}, nil
}

func (f *fakeRepository) UpdatePreset(ctx context.Context, id string, req PresetRequest) (CostPreset, error) {
	if id != plaID {
		return CostPreset{}, ErrPresetNotFound
	}
	return CostPreset{ID: id, Name: req.Name}, nil
}

func (f *fakeRepository) DeletePreset(ctx context.Context, id string) error {
	if id != plaID {
		return ErrPresetNotFound
	}
	return nil
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
	body := `{"batch_quantity":4,"sell_price":90,"items":[
		{"type":"material","kind":"fixed","name":"เส้น PLA","quantity":160,"unit":"กรัม","unit_price":0.45},
		{"type":"overhead","kind":"fixed","name":"ค่าเครื่อง","quantity":10,"unit":"ชม.","unit_price":6},
		{"type":"overhead","kind":"percent","name":"เผื่อเสีย","percent":10}]}`

	rec := sendRequest(newTestRouter(repo), http.MethodPut, "/api/v1/products/"+coffeeID+"/costs", body)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ต้องได้ %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	// 160 × 0.45 = 72, 10 × 6 = 60, 10% ของ 132 = 13.20 → รวม 145.20 ÷ 4 = 36.30
	wantAmounts := []float64{72, 60, 13.2}
	for i, want := range wantAmounts {
		if repo.savedAmounts[i] != want {
			t.Errorf("บรรทัดที่ %d = %v, ต้องได้ %v", i+1, repo.savedAmounts[i], want)
		}
	}
	if repo.savedCostPrice != 36.3 {
		t.Errorf("ต้นทุนต่อชิ้นที่บันทึก = %v, ต้องได้ 36.3", repo.savedCostPrice)
	}
	var got CostSheet
	json.NewDecoder(rec.Body).Decode(&got)
	if got.CostPrice != 36.3 || got.SellPrice != 90 || got.BatchQuantity != 4 {
		t.Errorf("ได้ %+v, ต้องได้ cost_price 36.3, sell_price 90, batch_quantity 4", got)
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
		{"ไม่บอกวิธีคิด ได้ 400", "/api/v1/products/" + coffeeID + "/costs", `{"batch_quantity":1,"items":[{"type":"material","name":"a","unit_price":1}]}`, http.StatusBadRequest, `{"error":"cost_kind_invalid"}`},
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

func TestHandler_Presets(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{"ดูรายการที่ใช้บ่อย ได้ 200", http.MethodGet, "/api/v1/cost-presets", "", http.StatusOK},
		{"เพิ่มรายการ ได้ 201", http.MethodPost, "/api/v1/cost-presets", `{"type":"material","kind":"fixed","name":"เส้น PLA","unit":"กรัม","unit_price":0.45}`, http.StatusCreated},
		{"เพิ่มรายการชื่อว่าง ได้ 400", http.MethodPost, "/api/v1/cost-presets", `{"type":"material","kind":"fixed","name":""}`, http.StatusBadRequest},
		{"แก้รายการ ได้ 200", http.MethodPut, "/api/v1/cost-presets/" + plaID, `{"type":"material","kind":"fixed","name":"เส้น PETG","unit_price":0.6}`, http.StatusOK},
		{"แก้รายการที่ไม่มี ได้ 404", http.MethodPut, "/api/v1/cost-presets/" + missingID, `{"type":"material","kind":"fixed","name":"a"}`, http.StatusNotFound},
		{"ลบรายการ ได้ 204", http.MethodDelete, "/api/v1/cost-presets/" + plaID, "", http.StatusNoContent},
		{"ลบรายการที่ไม่มี ได้ 404", http.MethodDelete, "/api/v1/cost-presets/" + missingID, "", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := coffeeRepo()
			repo.presets = []CostPreset{}
			rec := sendRequest(newTestRouter(repo), tt.method, tt.path, tt.body)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, ต้องได้ %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}
