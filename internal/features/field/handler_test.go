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
	err    error // ใส่ค่านี้เมื่ออยากจำลองว่า DB พัง
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
