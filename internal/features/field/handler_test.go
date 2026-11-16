package field

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
	colorID    = "11111111-1111-1111-1111-111111111111"
	missingID  = "99999999-9999-9999-9999-999999999999"
)

// fakeRepository ฐานข้อมูลปลอม เก็บช่องใน slice ใช้แทน Postgres ตอน test
type fakeRepository struct {
	fields []ProductField
	layout []LayoutItem // ลำดับที่บันทึกล่าสุด (nil = ยังไม่เคยบันทึก ใช้ค่าเริ่มต้น)
	err    error        // ใส่ค่านี้เมื่ออยากจำลองว่า DB พัง
}

func (f *fakeRepository) List(ctx context.Context) ([]ProductField, error) {
	return f.fields, f.err
}

func (f *fakeRepository) Create(ctx context.Context, req FieldRequest) (ProductField, error) {
	if f.err != nil {
		return ProductField{}, f.err
	}
	field := ProductField{ID: "new-id", Name: req.Name, Type: req.Type, Options: req.cleanOptions(), ShowInList: req.ShowInList}
	f.fields = append(f.fields, field)
	return field, nil
}

func (f *fakeRepository) Update(ctx context.Context, id string, req FieldRequest) (ProductField, error) {
	if f.err != nil {
		return ProductField{}, f.err
	}
	if id != colorID {
		return ProductField{}, ErrNotFound
	}
	return ProductField{ID: id, Name: req.Name, Type: req.Type, Options: req.cleanOptions()}, nil
}

func (f *fakeRepository) Delete(ctx context.Context, id string) error {
	if f.err != nil {
		return f.err
	}
	if id != colorID {
		return ErrNotFound
	}
	return nil
}

// Layout ลำดับช่องหลักตามค่าเริ่มต้น แล้วต่อด้วยช่องที่ร้านเพิ่ม (เหมือน DB จริงหลัง migration 000017)
func (f *fakeRepository) Layout(ctx context.Context) ([]LayoutItem, error) {
	if f.layout != nil || f.err != nil {
		return f.layout, f.err
	}
	items := []LayoutItem{}
	for i, block := range FormBlocks {
		items = append(items, LayoutItem{Key: block, SortOrder: i + 1})
	}
	for i, field := range f.fields {
		items = append(items, LayoutItem{Key: field.ID, SortOrder: len(FormBlocks) + i + 1})
	}
	return items, nil
}

// SaveLayout ช่องที่ส่งมาต้องครบพอดีเหมือน DB จริง (ช่องหลัก + ช่องที่มีอยู่) ไม่งั้น ErrLayoutInvalid
func (f *fakeRepository) SaveLayout(ctx context.Context, items []LayoutItem) error {
	if f.err != nil {
		return f.err
	}
	if len(items) != len(FormBlocks)+len(f.fields) {
		return ErrLayoutInvalid
	}
	f.layout = items
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

func TestHandler_RequiresLogin(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter(&fakeRepository{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/product-fields", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, ต้องได้ %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHandler_List(t *testing.T) {
	repo := &fakeRepository{fields: []ProductField{{ID: colorID, Name: "สี", Type: "select", Options: []string{"ขาว"}}}}
	rec := sendRequest(newTestRouter(repo), http.MethodGet, "/api/v1/product-fields", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ต้องได้ %d", rec.Code, http.StatusOK)
	}
	var got []ProductField
	json.NewDecoder(rec.Body).Decode(&got)
	if len(got) != 1 || got[0].Name != "สี" {
		t.Errorf("ได้ %+v, ต้องได้ช่อง สี 1 ช่อง", got)
	}
}

func TestHandler_List_DBError(t *testing.T) {
	rec := sendRequest(newTestRouter(&fakeRepository{err: errors.New("db down")}), http.MethodGet, "/api/v1/product-fields", "")

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, ต้องได้ %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{"ช่องตัวเลือก ได้ 201", `{"name":"สี","type":"select","options":["ขาว","ดำ"],"show_in_list":true}`, http.StatusCreated, ""},
		{"JSON ผิดรูปแบบ ได้ 400", `{`, http.StatusBadRequest, `{"error":"invalid_json"}`},
		{"ไม่มีชื่อ ได้ 400", `{"name":"","type":"text"}`, http.StatusBadRequest, `{"error":"field_name_required"}`},
		{"ชนิดผิด ได้ 400", `{"name":"สี","type":"x"}`, http.StatusBadRequest, `{"error":"field_type_invalid"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendRequest(newTestRouter(&fakeRepository{}), http.MethodPost, "/api/v1/product-fields", tt.body)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, ต้องได้ %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantBody != "" && rec.Body.String() != tt.wantBody {
				t.Errorf("body = %s, ต้องได้ %s", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestHandler_UpdateDelete(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		id         string
		wantStatus int
	}{
		{"แก้สำเร็จ ได้ 200", http.MethodPut, colorID, http.StatusOK},
		{"แก้ช่องที่ไม่มี ได้ 404", http.MethodPut, missingID, http.StatusNotFound},
		{"ลบสำเร็จ ได้ 204", http.MethodDelete, colorID, http.StatusNoContent},
		{"ลบช่องที่ไม่มี ได้ 404", http.MethodDelete, missingID, http.StatusNotFound},
		{"id ไม่ใช่ UUID ได้ 404", http.MethodDelete, "abc", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendRequest(newTestRouter(&fakeRepository{}), tt.method, "/api/v1/product-fields/"+tt.id, `{"name":"วัสดุ","type":"text"}`)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, ต้องได้ %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestHandler_Layout(t *testing.T) {
	repo := &fakeRepository{fields: []ProductField{{ID: colorID, Name: "สี", Type: "text"}}}
	rec := sendRequest(newTestRouter(repo), http.MethodGet, "/api/v1/product-form-layout", "")

	var got []LayoutItem
	json.NewDecoder(rec.Body).Decode(&got)
	if rec.Code != http.StatusOK || len(got) != 8 || got[0].Key != BlockImage || got[7].Key != colorID {
		t.Errorf("ต้องได้ช่องหลัก 7 + ช่องที่ร้านเพิ่ม 1 เรียงตามลำดับ ได้ %d %+v", rec.Code, got)
	}
}

// layoutBody ลำดับใหม่: ย้ายช่อง สี ขึ้นไปอยู่ใต้ชื่อสินค้า (เลข 3) ช่องหลักที่เหลือเลื่อนลง
func layoutBody(extra string) string {
	return `{"items":[{"key":"image","sort_order":1},{"key":"name","sort_order":2},{"key":"` + colorID + `","sort_order":3},` +
		`{"key":"code_category","sort_order":4},{"key":"unit","sort_order":5},{"key":"prices","sort_order":6},` +
		`{"key":"sale_stock","sort_order":7},{"key":"description","sort_order":8}` + extra + `]}`
}

func TestHandler_SaveLayout(t *testing.T) {
	repo := &fakeRepository{fields: []ProductField{{ID: colorID, Name: "สี", Type: "text"}}}
	rec := sendRequest(newTestRouter(repo), http.MethodPut, "/api/v1/product-form-layout", layoutBody(""))

	var got []LayoutItem
	json.NewDecoder(rec.Body).Decode(&got)
	if rec.Code != http.StatusOK || len(got) != 8 || got[2].Key != colorID || got[2].SortOrder != 3 {
		t.Errorf("บันทึกแล้วต้องได้ลำดับใหม่กลับมา ได้ %d %+v", rec.Code, got)
	}
}

func TestHandler_SaveLayout_Errors(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantCode int
		wantBody string
	}{
		{"JSON พัง", `{`, http.StatusBadRequest, `{"error":"invalid_json"}`},
		{"ขาดช่องหลัก", `{"items":[{"key":"image","sort_order":1}]}`, http.StatusBadRequest, `{"error":"form_layout_invalid"}`},
		{"ขาดช่องที่ร้านเพิ่ม (DB เช็ค)", strings.Replace(layoutBody(""), `{"key":"`+colorID+`","sort_order":3},`, "", 1), http.StatusBadRequest, `{"error":"form_layout_invalid"}`},
		{"มีช่องที่ไม่มีจริง (DB เช็ค)", layoutBody(`,{"key":"` + missingID + `","sort_order":9}`), http.StatusBadRequest, `{"error":"form_layout_invalid"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepository{fields: []ProductField{{ID: colorID, Name: "สี", Type: "text"}}}
			rec := sendRequest(newTestRouter(repo), http.MethodPut, "/api/v1/product-form-layout", tt.body)
			if rec.Code != tt.wantCode || rec.Body.String() != tt.wantBody {
				t.Errorf("ได้ %d %s ต้องได้ %d %s", rec.Code, rec.Body.String(), tt.wantCode, tt.wantBody)
			}
			if repo.layout != nil {
				t.Error("ผิดต้องไม่บันทึก")
			}
		})
	}
}
