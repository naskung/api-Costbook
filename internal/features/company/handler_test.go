package company

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/middleware"
	"github.com/naskung/api-Costbook/internal/utils"
)

const testSecret = "test-secret"

// fakeRepository ฐานข้อมูลปลอม เก็บข้อมูลร้าน 1 แถวไว้ในตัวแปร ใช้แทน Postgres ตอน test
type fakeRepository struct {
	company Company
	logo    Logo
	err     error // ใส่ค่านี้เมื่ออยากจำลองว่า DB พัง
}

func (f *fakeRepository) Get(ctx context.Context) (Company, error) {
	return f.company, f.err
}

func (f *fakeRepository) Update(ctx context.Context, req CompanyRequest) (Company, error) {
	if f.err != nil {
		return Company{}, f.err
	}
	f.company = Company{Name: req.Name, Address: req.Address, TaxID: req.TaxID, Branch: req.Branch, Phone: req.Phone, HasLogo: f.company.HasLogo}
	return f.company, nil
}

func (f *fakeRepository) FindLogo(ctx context.Context) (Logo, error) {
	if f.err != nil {
		return Logo{}, f.err
	}
	if f.logo.Data == nil {
		return Logo{}, ErrLogoNotFound
	}
	return f.logo, nil
}

func (f *fakeRepository) SaveLogo(ctx context.Context, logo Logo) (Company, error) {
	if f.err != nil {
		return Company{}, f.err
	}
	f.logo = logo
	f.company.HasLogo = true
	return f.company, nil
}

func (f *fakeRepository) DeleteLogo(ctx context.Context) error {
	if f.err != nil {
		return f.err
	}
	f.logo = Logo{}
	f.company.HasLogo = false
	return nil
}

// newTestRouter สร้าง Gin router สำหรับ test: route เหมือนของจริง แต่ใช้ fakeRepository แทน DB
func newTestRouter(repo Repository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler(repo).RegisterRoutes(r.Group("/api/v1"), middleware.Auth(testSecret))
	return r
}

// sendRequest ยิง request (ถ้า login = true จะแนบ token ไปด้วย)
func sendRequest(router *gin.Engine, req *http.Request, login bool) *httptest.ResponseRecorder {
	if login {
		token, _ := utils.CreateToken("user-1", testSecret, time.Hour)
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// newLogoRequest สร้าง request อัปโหลดไฟล์แบบ multipart/form-data
func newLogoRequest(t *testing.T, fieldName string, data []byte) *http.Request {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(fieldName, "logo.png")
	if err != nil {
		t.Fatalf("สร้าง form ไม่ได้: %v", err)
	}
	part.Write(data)
	writer.Close()

	req := httptest.NewRequest(http.MethodPut, "/api/v1/company/logo", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
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

func TestHandler_Get(t *testing.T) {
	repo := &fakeRepository{company: Company{Name: "ร้านตัวอย่าง", TaxID: "0105555555555", HasLogo: true}}
	router := newTestRouter(repo)

	// ไม่ต้อง login ก็ดูได้
	rec := sendRequest(router, httptest.NewRequest(http.MethodGet, "/api/v1/company", nil), false)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ต้องได้ %d", rec.Code, http.StatusOK)
	}
	var got Company
	json.NewDecoder(rec.Body).Decode(&got)
	if got != repo.company {
		t.Errorf("ได้ %+v, ต้องได้ %+v", got, repo.company)
	}
}

func TestHandler_Get_DBError(t *testing.T) {
	router := newTestRouter(&fakeRepository{err: errors.New("db down")})

	rec := sendRequest(router, httptest.NewRequest(http.MethodGet, "/api/v1/company", nil), false)

	checkError(t, rec, http.StatusInternalServerError, "server_error")
}

func TestHandler_Update(t *testing.T) {
	repo := &fakeRepository{}
	router := newTestRouter(repo)
	body := `{"name":"ร้านตัวอย่าง","address":"กรุงเทพฯ","tax_id":"0105555555555","branch":"สำนักงานใหญ่","phone":"02-123-4567"}`

	rec := sendRequest(router, httptest.NewRequest(http.MethodPut, "/api/v1/company", strings.NewReader(body)), true)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ต้องได้ %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got Company
	json.NewDecoder(rec.Body).Decode(&got)
	if got.Name != "ร้านตัวอย่าง" || got.TaxID != "0105555555555" || got.Phone != "02-123-4567" {
		t.Errorf("ได้ %+v, ต้องได้ข้อมูลที่ส่งไป", got)
	}
}

func TestHandler_Update_Fail(t *testing.T) {
	tests := []struct {
		name       string
		repo       *fakeRepository
		body       string
		login      bool
		wantStatus int
		wantError  string
	}{
		{"ไม่ได้ login ได้ 401", &fakeRepository{}, `{"name":"ร้าน"}`, false, http.StatusUnauthorized, "unauthorized"},
		{"JSON ผิดรูปแบบ ได้ 400", &fakeRepository{}, `{"name":`, true, http.StatusBadRequest, "invalid_json"},
		{"ชื่อว่าง ได้ 400", &fakeRepository{}, `{"name":""}`, true, http.StatusBadRequest, "company_name_required"},
		{"เลขภาษีผิด ได้ 400", &fakeRepository{}, `{"name":"ร้าน","tax_id":"123"}`, true, http.StatusBadRequest, "company_tax_id_invalid"},
		{"DB พัง ได้ 500", &fakeRepository{err: errors.New("db down")}, `{"name":"ร้าน"}`, true, http.StatusInternalServerError, "server_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/api/v1/company", strings.NewReader(tt.body))
			rec := sendRequest(newTestRouter(tt.repo), req, tt.login)

			checkError(t, rec, tt.wantStatus, tt.wantError)
		})
	}
}

func TestHandler_Logo(t *testing.T) {
	repo := &fakeRepository{logo: Logo{Data: []byte(pngBytes), ContentType: "image/png"}}
	router := newTestRouter(repo)

	// ไม่ต้อง login ก็ดูได้
	rec := sendRequest(router, httptest.NewRequest(http.MethodGet, "/api/v1/company/logo", nil), false)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ต้องได้ %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get("Content-Type") != "image/png" {
		t.Errorf("Content-Type = %q, ต้องได้ %q", rec.Header().Get("Content-Type"), "image/png")
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("ต้องมี header X-Content-Type-Options: nosniff")
	}
	if rec.Body.String() != pngBytes {
		t.Errorf("ได้ไฟล์ไม่ตรงกับที่เก็บไว้")
	}
}

func TestHandler_Logo_Fail(t *testing.T) {
	tests := []struct {
		name       string
		repo       *fakeRepository
		wantStatus int
		wantError  string
	}{
		{"ยังไม่มีโลโก้ ได้ 404", &fakeRepository{}, http.StatusNotFound, "logo_not_found"},
		{"DB พัง ได้ 500", &fakeRepository{err: errors.New("db down")}, http.StatusInternalServerError, "server_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/company/logo", nil)
			rec := sendRequest(newTestRouter(tt.repo), req, false)

			checkError(t, rec, tt.wantStatus, tt.wantError)
		})
	}
}

func TestHandler_UploadLogo(t *testing.T) {
	repo := &fakeRepository{company: Company{Name: "ร้านตัวอย่าง"}}
	router := newTestRouter(repo)

	rec := sendRequest(router, newLogoRequest(t, "logo", []byte(pngBytes)), true)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ต้องได้ %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got Company
	json.NewDecoder(rec.Body).Decode(&got)
	if !got.HasLogo {
		t.Error("has_logo = false, ต้องได้ true")
	}
	if string(repo.logo.Data) != pngBytes || repo.logo.ContentType != "image/png" {
		t.Errorf("โลโก้ที่บันทึก = %q (%s), ต้องได้ไฟล์ PNG ที่ส่งไป", repo.logo.Data, repo.logo.ContentType)
	}
}

func TestHandler_UploadLogo_Fail(t *testing.T) {
	// ไฟล์ PNG ที่ใหญ่เกิน 1 MB นิดเดียว และใหญ่เกินมาก
	justTooLarge := append([]byte(pngBytes), make([]byte, maxLogoSize)...)
	wayTooLarge := append([]byte(pngBytes), make([]byte, 5*maxLogoSize)...)

	tests := []struct {
		name       string
		repo       *fakeRepository
		req        *http.Request
		login      bool
		wantStatus int
		wantError  string
	}{
		{"ไม่ได้ login ได้ 401", &fakeRepository{}, newLogoRequest(t, "logo", []byte(pngBytes)), false, http.StatusUnauthorized, "unauthorized"},
		{"ไม่ได้แนบไฟล์ ได้ 400", &fakeRepository{}, httptest.NewRequest(http.MethodPut, "/api/v1/company/logo", nil), true, http.StatusBadRequest, "logo_required"},
		{"ชื่อช่องผิด ได้ 400", &fakeRepository{}, newLogoRequest(t, "image", []byte(pngBytes)), true, http.StatusBadRequest, "logo_required"},
		{"ไฟล์ว่าง ได้ 400", &fakeRepository{}, newLogoRequest(t, "logo", []byte{}), true, http.StatusBadRequest, "logo_required"},
		{"ใหญ่เกิน 1 MB ได้ 400", &fakeRepository{}, newLogoRequest(t, "logo", justTooLarge), true, http.StatusBadRequest, "logo_too_large"},
		{"ใหญ่เกินมาก ได้ 400", &fakeRepository{}, newLogoRequest(t, "logo", wayTooLarge), true, http.StatusBadRequest, "logo_too_large"},
		{"ไฟล์ GIF ได้ 400", &fakeRepository{}, newLogoRequest(t, "logo", []byte(gifBytes)), true, http.StatusBadRequest, "logo_type_invalid"},
		{"DB พัง ได้ 500", &fakeRepository{err: errors.New("db down")}, newLogoRequest(t, "logo", []byte(pngBytes)), true, http.StatusInternalServerError, "server_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendRequest(newTestRouter(tt.repo), tt.req, tt.login)

			checkError(t, rec, tt.wantStatus, tt.wantError)
		})
	}
}

func TestHandler_DeleteLogo(t *testing.T) {
	repo := &fakeRepository{company: Company{HasLogo: true}, logo: Logo{Data: []byte(pngBytes), ContentType: "image/png"}}
	router := newTestRouter(repo)

	rec := sendRequest(router, httptest.NewRequest(http.MethodDelete, "/api/v1/company/logo", nil), true)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, ต้องได้ %d", rec.Code, http.StatusNoContent)
	}
	body, _ := io.ReadAll(rec.Body)
	if len(body) != 0 {
		t.Errorf("body = %s, ต้องว่าง", body)
	}
	if repo.logo.Data != nil || repo.company.HasLogo {
		t.Error("โลโก้ต้องถูกลบ")
	}
}

func TestHandler_DeleteLogo_Fail(t *testing.T) {
	tests := []struct {
		name       string
		repo       *fakeRepository
		login      bool
		wantStatus int
		wantError  string
	}{
		{"ไม่ได้ login ได้ 401", &fakeRepository{}, false, http.StatusUnauthorized, "unauthorized"},
		{"DB พัง ได้ 500", &fakeRepository{err: errors.New("db down")}, true, http.StatusInternalServerError, "server_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/company/logo", nil)
			rec := sendRequest(newTestRouter(tt.repo), req, tt.login)

			checkError(t, rec, tt.wantStatus, tt.wantError)
		})
	}
}
