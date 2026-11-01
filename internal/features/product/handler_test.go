package product

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	products  []Product
	inUseID   string // id สินค้าที่จำลองว่าเคยอยู่ในใบกำกับ
	image     Image  // รูปของสินค้า (ใช้รูปเดียวทั้ง fake)
	gotFilter ListFilter
	err       error // ใส่ค่านี้เมื่ออยากจำลองว่า DB พัง
}

func (f *fakeRepository) List(ctx context.Context, filter ListFilter) ([]Product, error) {
	f.gotFilter = filter
	return f.products, f.err
}

func (f *fakeRepository) Counts(ctx context.Context) (Counts, error) {
	return Counts{All: len(f.products), ForSale: len(f.products)}, f.err
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
	for _, p := range f.products {
		if req.Code != "" && p.Code == req.Code {
			return Product{}, ErrCodeTaken
		}
	}
	if req.categoryIDText() == missingID {
		return Product{}, ErrCategoryNotFound
	}
	p := Product{
		ID: "new-id", Code: req.Code, Name: req.Name, CategoryID: req.categoryIDText(), Unit: req.Unit, CostPrice: req.CostPrice, SellPrice: req.SellPrice,
		MinStock: req.MinStock, BatchQuantity: 1, TrackStock: req.trackStock(), CustomFields: req.customFields(),
	}
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
	if req.TrackStock != nil {
		p.TrackStock = *req.TrackStock
	}
	return p, nil
}

func (f *fakeRepository) Delete(ctx context.Context, id string) error {
	if id == f.inUseID {
		return ErrInUse
	}
	_, err := f.FindByID(ctx, id)
	return err
}

func (f *fakeRepository) FindImage(ctx context.Context, id string) (Image, error) {
	if _, err := f.FindByID(ctx, id); err != nil {
		return Image{}, err
	}
	if len(f.image.Data) == 0 {
		return Image{}, ErrImageNotFound
	}
	return f.image, nil
}

func (f *fakeRepository) SaveImage(ctx context.Context, id string, image Image) (Product, error) {
	p, err := f.FindByID(ctx, id)
	if err != nil {
		return Product{}, err
	}
	f.image = image
	p.ImageVersion = 1700000000000
	return p, nil
}

func (f *fakeRepository) DeleteImage(ctx context.Context, id string) (Product, error) {
	p, err := f.FindByID(ctx, id)
	if err != nil {
		return Product{}, err
	}
	f.image = Image{}
	p.ImageVersion = 0
	return p, nil
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
	return &fakeRepository{products: []Product{{ID: coffeeID, Code: "COF-01", Name: "กาแฟ", Unit: "แก้ว", SellPrice: 45, BatchQuantity: 1}}}
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

func TestHandler_List_Filter(t *testing.T) {
	const drinkID = "5b8f0d4e-3c1a-4f7e-9a2b-6d1c8e0f7a3b"
	repo := coffeeRepo()
	path := "/api/v1/products?q=" + url.QueryEscape("กาแฟ") + "&category_id=" + drinkID + "&kind=sale&low=true&limit=20&offset=20"
	rec := sendRequest(newTestRouter(repo), http.MethodGet, path, "")
	want := ListFilter{Query: "กาแฟ", CategoryID: drinkID, Kind: KindForSale, OnlyLow: true, Limit: 20, Offset: 20}
	if rec.Code != http.StatusOK || repo.gotFilter != want {
		t.Errorf("ต้องส่งตัวกรองต่อให้ repository ได้ %d %+v", rec.Code, repo.gotFilter)
	}
}

func TestHandler_List_BadFilter(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"kind ไม่มีจริง", "/api/v1/products?kind=x", `{"error":"product_kind_invalid"}`},
		{"category_id ไม่ใช่ UUID", "/api/v1/products?category_id=abc", `{"error":"product_category_invalid"}`},
		{"limit ผิด", "/api/v1/products?limit=-5", `{"error":"page_invalid"}`},
	}
	for _, tt := range tests {
		rec := sendRequest(newTestRouter(coffeeRepo()), http.MethodGet, tt.path, "")
		if rec.Code != http.StatusBadRequest || rec.Body.String() != tt.want {
			t.Errorf("%s: ได้ %d %s ต้องได้ 400 %s", tt.name, rec.Code, rec.Body.String(), tt.want)
		}
	}
}

func TestHandler_Counts(t *testing.T) {
	rec := sendRequest(newTestRouter(coffeeRepo()), http.MethodGet, "/api/v1/products/counts", "")
	if rec.Code != http.StatusOK || rec.Body.String() != `{"all":1,"for_sale":1,"material":0,"low":0}` {
		t.Errorf("ได้ %d %s", rec.Code, rec.Body.String())
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

func TestHandler_Create_NewFields(t *testing.T) {
	body := `{"name":"ค่าออกแบบโมเดล","code":"SV-01","category":"บริการ","track_stock":false,"custom_fields":{"f1":"งานด่วน"}}`
	rec := sendRequest(newTestRouter(coffeeRepo()), http.MethodPost, "/api/v1/products", body)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, ต้องได้ %d (body: %s)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var got Product
	json.NewDecoder(rec.Body).Decode(&got)
	if got.TrackStock || got.Code != "SV-01" || got.CustomFields["f1"] != "งานด่วน" {
		t.Errorf("ได้ %+v, ต้องได้ track_stock false, code SV-01, custom_fields f1", got)
	}
}

func TestHandler_Create_DefaultTrackStock(t *testing.T) {
	rec := sendRequest(newTestRouter(coffeeRepo()), http.MethodPost, "/api/v1/products", `{"name":"ชาไทย"}`)

	var got Product
	json.NewDecoder(rec.Body).Decode(&got)
	if !got.TrackStock {
		t.Errorf("ไม่ส่ง track_stock มา ได้ %v, ต้องได้ true (นับสต็อก)", got.TrackStock)
	}
	if got.CustomFields == nil {
		t.Errorf("ไม่ส่ง custom_fields มา ต้องได้ {} ไม่ใช่ null")
	}
}

func TestHandler_Create_CodeTaken(t *testing.T) {
	rec := sendRequest(newTestRouter(coffeeRepo()), http.MethodPost, "/api/v1/products", `{"name":"กาแฟร้อน","code":"COF-01"}`)

	checkError(t, rec, http.StatusConflict, "product_code_taken")
}

const testPNG = "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"

// sendImage อัปโหลดรูปแบบ multipart ช่อง "image" พร้อม token
func sendImage(t *testing.T, router *gin.Engine, path string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("image", "case.png")
	if err != nil {
		t.Fatal(err)
	}
	part.Write(data)
	writer.Close()

	req := httptest.NewRequest(http.MethodPut, path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	token, _ := utils.CreateToken("user-1", testSecret, time.Hour)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestHandler_UploadImage(t *testing.T) {
	repo := coffeeRepo()
	rec := sendImage(t, newTestRouter(repo), "/api/v1/products/"+coffeeID+"/image", []byte(testPNG))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ต้องได้ %d (%s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got Product
	json.NewDecoder(rec.Body).Decode(&got)
	if got.ImageVersion == 0 {
		t.Error("image_version = 0, ต้องได้เวลาที่เปลี่ยนรูป")
	}
	if string(repo.image.Data) != testPNG || repo.image.ContentType != "image/png" {
		t.Errorf("รูปที่บันทึก = %q (%s), ต้องได้ไฟล์ PNG ที่ส่งไป", repo.image.Data, repo.image.ContentType)
	}
}

func TestHandler_UploadImage_Fail(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		data       []byte
		wantStatus int
		wantError  string
	}{
		{"ไม่ใช่รูป ได้ 400", "/api/v1/products/" + coffeeID + "/image", []byte("hello"), http.StatusBadRequest, "product_image_type_invalid"},
		{"ใหญ่เกิน 1 MB ได้ 400", "/api/v1/products/" + coffeeID + "/image", append([]byte(testPNG), make([]byte, 1<<20)...), http.StatusBadRequest, "product_image_too_large"},
		{"ไฟล์ว่าง ได้ 400", "/api/v1/products/" + coffeeID + "/image", []byte{}, http.StatusBadRequest, "product_image_required"},
		{"ไม่มีสินค้า ได้ 404", "/api/v1/products/" + missingID + "/image", []byte(testPNG), http.StatusNotFound, "product_not_found"},
		{"id ไม่ใช่ UUID ได้ 404", "/api/v1/products/abc/image", []byte(testPNG), http.StatusNotFound, "product_not_found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendImage(t, newTestRouter(coffeeRepo()), tt.path, tt.data)
			checkError(t, rec, tt.wantStatus, tt.wantError)
		})
	}
}

func TestHandler_Image(t *testing.T) {
	repo := coffeeRepo()
	repo.image = Image{Data: []byte(testPNG), ContentType: "image/png"}
	rec := sendRequest(newTestRouter(repo), http.MethodGet, "/api/v1/products/"+coffeeID+"/image?v=1", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ต้องได้ %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get("Content-Type") != "image/png" || rec.Body.String() != testPNG {
		t.Errorf("ได้ %s %q, ต้องได้รูป PNG", rec.Header().Get("Content-Type"), rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Cache-Control"), "max-age") {
		t.Errorf("Cache-Control = %q, ต้องให้ browser เก็บรูปไว้ได้", rec.Header().Get("Cache-Control"))
	}
}

func TestHandler_Image_Fail(t *testing.T) {
	tests := []struct {
		name      string
		repo      *fakeRepository
		path      string
		wantError string
	}{
		{"ยังไม่มีรูป ได้ 404", coffeeRepo(), "/api/v1/products/" + coffeeID + "/image", "product_image_not_found"},
		{"ไม่มีสินค้า ได้ 404", coffeeRepo(), "/api/v1/products/" + missingID + "/image", "product_not_found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendRequest(newTestRouter(tt.repo), http.MethodGet, tt.path, "")
			checkError(t, rec, http.StatusNotFound, tt.wantError)
		})
	}
}

func TestHandler_Image_RequiresLogin(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products/"+coffeeID+"/image", nil)
	rec := httptest.NewRecorder()
	newTestRouter(coffeeRepo()).ServeHTTP(rec, req)

	checkError(t, rec, http.StatusUnauthorized, "unauthorized")
}

func TestHandler_DeleteImage(t *testing.T) {
	repo := coffeeRepo()
	repo.image = Image{Data: []byte(testPNG), ContentType: "image/png"}
	rec := sendRequest(newTestRouter(repo), http.MethodDelete, "/api/v1/products/"+coffeeID+"/image", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ต้องได้ %d", rec.Code, http.StatusOK)
	}
	var got Product
	json.NewDecoder(rec.Body).Decode(&got)
	if got.ImageVersion != 0 || len(repo.image.Data) != 0 {
		t.Errorf("image_version = %d, รูปเหลือ %d ไบต์ ต้องได้ 0 ทั้งคู่", got.ImageVersion, len(repo.image.Data))
	}
}

func TestHandler_DeleteImage_NotFound(t *testing.T) {
	rec := sendRequest(newTestRouter(coffeeRepo()), http.MethodDelete, "/api/v1/products/"+missingID+"/image", "")
	checkError(t, rec, http.StatusNotFound, "product_not_found")
}

func TestHandler_ImageDBError(t *testing.T) {
	repo := coffeeRepo()
	repo.err = errors.New("db down")
	router := newTestRouter(repo)

	checkError(t, sendRequest(router, http.MethodGet, "/api/v1/products/"+coffeeID+"/image", ""), http.StatusInternalServerError, "server_error")
	checkError(t, sendRequest(router, http.MethodDelete, "/api/v1/products/"+coffeeID+"/image", ""), http.StatusInternalServerError, "server_error")
	checkError(t, sendImage(t, router, "/api/v1/products/"+coffeeID+"/image", []byte(testPNG)), http.StatusInternalServerError, "server_error")
}

func TestHandler_Create_Category(t *testing.T) {
	router := newTestRouter(coffeeRepo())
	categoryID := "c1111111-1111-1111-1111-111111111111"

	rec := sendRequest(router, http.MethodPost, "/api/v1/products", `{"name":"เคส","category_id":"`+categoryID+`"}`)
	var got Product
	json.NewDecoder(rec.Body).Decode(&got)
	if rec.Code != http.StatusCreated || got.CategoryID != categoryID {
		t.Errorf("ได้ %d category_id = %q, ต้องได้ 201 และ category_id ที่ส่งไป", rec.Code, got.CategoryID)
	}

	checkError(t, sendRequest(router, http.MethodPost, "/api/v1/products", `{"name":"เคส","category_id":"งานพิมพ์"}`), http.StatusBadRequest, "product_category_invalid")
	checkError(t, sendRequest(router, http.MethodPost, "/api/v1/products", `{"name":"เคส","category_id":"`+missingID+`"}`), http.StatusBadRequest, "product_category_invalid")
}
