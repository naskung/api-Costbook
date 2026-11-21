package quotation

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
	testSecret  = "test-secret"
	quotationID = "11111111-1111-1111-1111-111111111111"
	missingID   = "99999999-9999-9999-9999-999999999999"
	coffeeID    = "33333333-3333-3333-3333-333333333333"
	validBody   = `{"issue_date":"2026-11-21","valid_until":"2026-12-21","customer_name":"บริษัท ลูกค้า จำกัด","items":[{"product_id":"` + coffeeID + `","quantity":2,"unit_price":50}]}`
)

// fakeRepository ฐานข้อมูลปลอม ใช้แทน Postgres ตอน test
// createErr / cancelErr ใช้จำลอง error ของ DB เช่น สินค้าเป็นวัสดุ ใบยกเลิกไปแล้ว
type fakeRepository struct {
	quotation Quotation
	gotPeriod utils.DateRange
	gotPage   [2]int
	createErr error
	cancelErr error
	err       error
}

func (f *fakeRepository) List(ctx context.Context, period utils.DateRange, limit, offset int) ([]QuotationSummary, error) {
	f.gotPeriod = period
	f.gotPage = [2]int{limit, offset}
	return []QuotationSummary{}, f.err
}

func (f *fakeRepository) FindByID(ctx context.Context, id string) (Quotation, error) {
	if f.err != nil {
		return Quotation{}, f.err
	}
	if id != f.quotation.ID {
		return Quotation{}, ErrNotFound
	}
	return f.quotation, nil
}

func (f *fakeRepository) Create(ctx context.Context, req QuotationRequest) (Quotation, error) {
	if f.createErr != nil {
		return Quotation{}, f.createErr
	}
	m := utils.DocumentAmounts(itemsTotal(req.Items), req.Discount, true, false)
	return Quotation{
		ID: "new-id", QuotationNo: quotationNumber(2026, 1), Status: "issued", ValidUntil: req.ValidUntil,
		CustomerName: req.CustomerName, WithVAT: true, ItemsTotal: m.ItemsTotal, Discount: m.Discount,
		Subtotal: m.Subtotal, VAT: m.VAT, Total: m.Total,
	}, nil
}

func (f *fakeRepository) Cancel(ctx context.Context, id string) (Quotation, error) {
	if f.cancelErr != nil {
		return Quotation{}, f.cancelErr
	}
	q, err := f.FindByID(ctx, id)
	q.Status = "cancelled"
	return q, err
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

func issuedRepo() *fakeRepository {
	return &fakeRepository{quotation: Quotation{ID: quotationID, QuotationNo: "QT-2026-0001", Status: "issued", Items: []QuotationItem{}}}
}

func TestHandler_RequiresLogin(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter(issuedRepo()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/quotations", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, ต้องได้ %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHandler_List(t *testing.T) {
	repo := issuedRepo()
	rec := sendRequest(newTestRouter(repo), http.MethodGet, "/api/v1/quotations?from=2026-11-01&to=2026-11-21&limit=20&offset=40", "")

	if rec.Code != http.StatusOK || rec.Body.String() != "[]" {
		t.Errorf("status = %d body = %s, ต้องได้ 200 []", rec.Code, rec.Body.String())
	}
	if repo.gotPeriod != (utils.DateRange{From: "2026-11-01", To: "2026-11-21"}) {
		t.Errorf("period = %+v, ต้องได้ 2026-11-01 ถึง 2026-11-21", repo.gotPeriod)
	}
	if repo.gotPage != [2]int{20, 40} {
		t.Errorf("ต้องส่ง limit / offset ต่อให้ repository ได้ %v", repo.gotPage)
	}
}

func TestHandler_List_BadParams(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantBody string
	}{
		{"from หลัง to ได้ date_range_invalid", "?from=2026-11-21&to=2026-11-01", `{"error":"date_range_invalid"}`},
		{"limit ไม่ใช่ตัวเลข ได้ page_invalid", "?limit=abc", `{"error":"page_invalid"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendRequest(newTestRouter(issuedRepo()), http.MethodGet, "/api/v1/quotations"+tt.query, "")
			if rec.Code != http.StatusBadRequest || rec.Body.String() != tt.wantBody {
				t.Errorf("ได้ %d %s, ต้องได้ 400 %s", rec.Code, rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		repo       *fakeRepository
		id         string
		wantStatus int
	}{
		{"เจอใบเสนอราคา ได้ 200", issuedRepo(), quotationID, http.StatusOK},
		{"ไม่มีใบนี้ ได้ 404", issuedRepo(), missingID, http.StatusNotFound},
		{"id ไม่ใช่ UUID ได้ 404", issuedRepo(), "QT-2026-0001", http.StatusNotFound},
		{"DB พัง ได้ 500", &fakeRepository{err: errors.New("db down")}, quotationID, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendRequest(newTestRouter(tt.repo), http.MethodGet, "/api/v1/quotations/"+tt.id, "")
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, ต้องได้ %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandler_Create(t *testing.T) {
	rec := sendRequest(newTestRouter(issuedRepo()), http.MethodPost, "/api/v1/quotations", validBody)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, ต้องได้ %d (body: %s)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var got Quotation
	json.NewDecoder(rec.Body).Decode(&got)
	if got.QuotationNo != "QT-2026-0001" || got.ValidUntil != "2026-12-21" {
		t.Errorf("ได้เลขที่ %q ยืนราคาถึง %q, ต้องได้ QT-2026-0001 / 2026-12-21", got.QuotationNo, got.ValidUntil)
	}
	if got.Subtotal != 100 || got.VAT != 7 || got.Total != 107 {
		t.Errorf("ได้ ยอด %v VAT %v รวม %v, ต้องได้ 100 / 7 / 107", got.Subtotal, got.VAT, got.Total)
	}
}

func TestHandler_Create_Errors(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		createErr  error
		wantStatus int
		wantBody   string
	}{
		{"JSON ผิดรูปแบบ ได้ 400", `{`, nil, http.StatusBadRequest, `{"error":"invalid_json"}`},
		{"ไม่มีชื่อลูกค้า ได้ 400", `{"issue_date":"2026-11-21","valid_until":"2026-12-21","items":[{"product_id":"` + coffeeID + `","quantity":1,"unit_price":50}]}`, nil, http.StatusBadRequest, `{"error":"quotation_customer_required"}`},
		{"ยืนราคาถึงก่อนวันที่ออก ได้ 400", `{"issue_date":"2026-11-21","valid_until":"2026-11-01","customer_name":"ลูกค้า","items":[]}`, nil, http.StatusBadRequest, `{"error":"quotation_valid_until_invalid"}`},
		{"ไม่มีสินค้าในใบ ได้ 400", `{"issue_date":"2026-11-21","valid_until":"2026-12-21","customer_name":"ลูกค้า","items":[]}`, nil, http.StatusBadRequest, `{"error":"quotation_items_required"}`},
		{"สินค้าเป็นวัสดุ (ไม่ขาย) ได้ 409", validBody, ErrNotForSale, http.StatusConflict, `{"error":"product_not_for_sale"}`},
		{"ไม่มีสินค้านี้ ได้ 404", validBody, ErrProductNotFound, http.StatusNotFound, `{"error":"product_not_found"}`},
		{"DB พัง ได้ 500", validBody, errors.New("db down"), http.StatusInternalServerError, `{"error":"server_error"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := issuedRepo()
			repo.createErr = tt.createErr
			rec := sendRequest(newTestRouter(repo), http.MethodPost, "/api/v1/quotations", tt.body)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, ต้องได้ %d", rec.Code, tt.wantStatus)
			}
			if rec.Body.String() != tt.wantBody {
				t.Errorf("body = %s, ต้องได้ %s", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestHandler_Cancel(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		cancelErr      error
		wantStatus     int
		wantStatusText string
	}{
		{"ยกเลิกสำเร็จ ได้ 200 สถานะ cancelled", quotationID, nil, http.StatusOK, "cancelled"},
		{"ยกเลิกซ้ำ ได้ 409", quotationID, ErrAlreadyCancelled, http.StatusConflict, ""},
		{"ไม่มีใบนี้ ได้ 404", missingID, ErrNotFound, http.StatusNotFound, ""},
		{"id ไม่ใช่ UUID ได้ 404", "abc", nil, http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := issuedRepo()
			repo.cancelErr = tt.cancelErr
			rec := sendRequest(newTestRouter(repo), http.MethodPost, "/api/v1/quotations/"+tt.id+"/cancel", "")
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, ต้องได้ %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatusText == "" {
				return
			}
			var got Quotation
			json.NewDecoder(rec.Body).Decode(&got)
			if got.Status != tt.wantStatusText {
				t.Errorf("status ของใบ = %q, ต้องได้ %q", got.Status, tt.wantStatusText)
			}
		})
	}
}
