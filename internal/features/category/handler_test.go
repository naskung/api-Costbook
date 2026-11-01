package category

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
	printID    = "11111111-1111-1111-1111-111111111111"
	missingID  = "99999999-9999-9999-9999-999999999999"
)

// fakeRepository ฐานข้อมูลปลอม เก็บหมวดใน slice ใช้แทน Postgres ตอน test
type fakeRepository struct {
	categories []Category
	err        error // ใส่ค่านี้เมื่ออยากจำลองว่า DB พัง
}

func (f *fakeRepository) List(ctx context.Context) ([]Category, error) {
	return f.categories, f.err
}

func (f *fakeRepository) Create(ctx context.Context, req CategoryRequest) (Category, error) {
	if f.err != nil {
		return Category{}, f.err
	}
	for _, c := range f.categories {
		if strings.EqualFold(c.Name, req.cleanName()) {
			return Category{}, ErrNameTaken
		}
	}
	c := Category{ID: "new-id", Name: req.cleanName()}
	f.categories = append(f.categories, c)
	return c, nil
}

func (f *fakeRepository) Update(ctx context.Context, id string, req CategoryRequest) (Category, error) {
	if f.err != nil {
		return Category{}, f.err
	}
	for i, c := range f.categories {
		if c.ID == id {
			f.categories[i].Name = req.cleanName()
			return f.categories[i], nil
		}
	}
	return Category{}, ErrNotFound
}

func (f *fakeRepository) Delete(ctx context.Context, id string) error {
	if f.err != nil {
		return f.err
	}
	for i, c := range f.categories {
		if c.ID == id {
			f.categories = append(f.categories[:i], f.categories[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func printRepo() *fakeRepository {
	return &fakeRepository{categories: []Category{{ID: printID, Name: "งานพิมพ์ 3D", ProductCount: 2}}}
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

func checkError(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantError string) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Errorf("status = %d, ต้องได้ %d", rec.Code, wantStatus)
	}
	if want := `{"error":"` + wantError + `"}`; rec.Body.String() != want {
		t.Errorf("body = %s, ต้องได้ %s", rec.Body.String(), want)
	}
}

func TestHandler_RequiresLogin(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/product-categories", nil)
	rec := httptest.NewRecorder()
	newTestRouter(printRepo()).ServeHTTP(rec, req)
	checkError(t, rec, http.StatusUnauthorized, "unauthorized")
}

func TestHandler_List(t *testing.T) {
	rec := sendRequest(newTestRouter(printRepo()), http.MethodGet, "/api/v1/product-categories", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ต้องได้ %d", rec.Code, http.StatusOK)
	}
	var got []Category
	json.NewDecoder(rec.Body).Decode(&got)
	if len(got) != 1 || got[0].Name != "งานพิมพ์ 3D" || got[0].ProductCount != 2 {
		t.Errorf("ได้ %+v, ต้องได้หมวด งานพิมพ์ 3D มีสินค้า 2 ตัว", got)
	}
}

func TestHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{"ชื่อใหม่ ได้ 201", `{"name":"  พวงกุญแจ "}`, http.StatusCreated, ""},
		{"ชื่อซ้ำ (ตัวพิมพ์ต่างกัน) ได้ 409", `{"name":"งานพิมพ์ 3d"}`, http.StatusConflict, "category_name_taken"},
		{"ชื่อว่าง ได้ 400", `{"name":" "}`, http.StatusBadRequest, "category_name_required"},
		{"JSON ผิด ได้ 400", `{`, http.StatusBadRequest, "invalid_json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendRequest(newTestRouter(printRepo()), http.MethodPost, "/api/v1/product-categories", tt.body)
			if tt.wantError != "" {
				checkError(t, rec, tt.wantStatus, tt.wantError)
				return
			}
			var got Category
			json.NewDecoder(rec.Body).Decode(&got)
			if rec.Code != tt.wantStatus || got.Name != "พวงกุญแจ" {
				t.Errorf("ได้ %d %+v, ต้องได้ %d ชื่อ พวงกุญแจ (ตัดช่องว่างแล้ว)", rec.Code, got, tt.wantStatus)
			}
		})
	}
}

func TestHandler_Update(t *testing.T) {
	rec := sendRequest(newTestRouter(printRepo()), http.MethodPut, "/api/v1/product-categories/"+printID, `{"name":"งานพิมพ์สามมิติ"}`)

	var got Category
	json.NewDecoder(rec.Body).Decode(&got)
	if rec.Code != http.StatusOK || got.Name != "งานพิมพ์สามมิติ" || got.ProductCount != 2 {
		t.Errorf("ได้ %d %+v, ต้องได้ชื่อใหม่ และสินค้ายังอยู่ 2 ตัว", rec.Code, got)
	}
}

func TestHandler_Update_Fail(t *testing.T) {
	router := newTestRouter(printRepo())
	checkError(t, sendRequest(router, http.MethodPut, "/api/v1/product-categories/"+missingID, `{"name":"ใหม่"}`), http.StatusNotFound, "category_not_found")
	checkError(t, sendRequest(router, http.MethodPut, "/api/v1/product-categories/abc", `{"name":"ใหม่"}`), http.StatusNotFound, "category_not_found")
	checkError(t, sendRequest(router, http.MethodPut, "/api/v1/product-categories/"+printID, `{"name":""}`), http.StatusBadRequest, "category_name_required")
}

func TestHandler_Delete(t *testing.T) {
	repo := printRepo()
	rec := sendRequest(newTestRouter(repo), http.MethodDelete, "/api/v1/product-categories/"+printID, "")

	if rec.Code != http.StatusNoContent || len(repo.categories) != 0 {
		t.Errorf("status = %d, เหลือ %d หมวด ต้องได้ 204 และไม่เหลือหมวด", rec.Code, len(repo.categories))
	}
	checkError(t, sendRequest(newTestRouter(repo), http.MethodDelete, "/api/v1/product-categories/"+printID, ""), http.StatusNotFound, "category_not_found")
}

func TestHandler_DBError(t *testing.T) {
	router := newTestRouter(&fakeRepository{err: errors.New("db down")})
	checkError(t, sendRequest(router, http.MethodGet, "/api/v1/product-categories", ""), http.StatusInternalServerError, "server_error")
	checkError(t, sendRequest(router, http.MethodPost, "/api/v1/product-categories", `{"name":"ใหม่"}`), http.StatusInternalServerError, "server_error")
}
