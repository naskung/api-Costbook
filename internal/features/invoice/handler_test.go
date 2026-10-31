package invoice

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
	invoiceID  = "11111111-1111-1111-1111-111111111111"
	missingID  = "99999999-9999-9999-9999-999999999999"
	coffeeID   = "33333333-3333-3333-3333-333333333333"
	validBody  = `{"issue_date":"2026-10-31","customer_name":"บริษัท ลูกค้า จำกัด","items":[{"product_id":"` + coffeeID + `","quantity":2,"unit_price":50}]}`
)

// fakeRepository ฐานข้อมูลปลอม ใช้แทน Postgres ตอน test
// createErr / cancelErr ใช้จำลอง error ของ DB เช่น ของไม่พอ ร้านไม่มีเลขภาษี
type fakeRepository struct {
	invoice       Invoice
	gotMonthStart string
	createErr     error
	cancelErr     error
	err           error
}

func (f *fakeRepository) List(ctx context.Context, monthStart string) ([]InvoiceSummary, error) {
	f.gotMonthStart = monthStart
	return []InvoiceSummary{}, f.err
}

func (f *fakeRepository) FindByID(ctx context.Context, id string) (Invoice, error) {
	if f.err != nil {
		return Invoice{}, f.err
	}
	if id != f.invoice.ID {
		return Invoice{}, ErrNotFound
	}
	return f.invoice, nil
}

func (f *fakeRepository) Create(ctx context.Context, req InvoiceRequest) (Invoice, error) {
	if f.createErr != nil {
		return Invoice{}, f.createErr
	}
	subtotal, vat, total := totals(req.Items)
	return Invoice{ID: "new-id", InvoiceNo: invoiceNumber(2026, 1), Status: "issued", CustomerName: req.CustomerName, Subtotal: subtotal, VAT: vat, Total: total}, nil
}

func (f *fakeRepository) Cancel(ctx context.Context, id string) (Invoice, error) {
	if f.cancelErr != nil {
		return Invoice{}, f.cancelErr
	}
	inv, err := f.FindByID(ctx, id)
	inv.Status = "cancelled"
	return inv, err
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
	return &fakeRepository{invoice: Invoice{ID: invoiceID, InvoiceNo: "INV-2026-0001", Status: "issued", Items: []InvoiceItem{}}}
}

func TestHandler_RequiresLogin(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter(issuedRepo()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/invoices", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, ต้องได้ %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHandler_List(t *testing.T) {
	repo := issuedRepo()
	rec := sendRequest(newTestRouter(repo), http.MethodGet, "/api/v1/invoices?month=2026-10", "")

	if rec.Code != http.StatusOK || rec.Body.String() != "[]" {
		t.Errorf("status = %d body = %s, ต้องได้ 200 []", rec.Code, rec.Body.String())
	}
	if repo.gotMonthStart != "2026-10-01" {
		t.Errorf("monthStart = %q, ต้องได้ 2026-10-01", repo.gotMonthStart)
	}
}

func TestHandler_List_BadMonth(t *testing.T) {
	rec := sendRequest(newTestRouter(issuedRepo()), http.MethodGet, "/api/v1/invoices?month=2026", "")

	if rec.Body.String() != `{"error":"month_invalid"}` {
		t.Errorf("body = %s, ต้องได้ month_invalid", rec.Body.String())
	}
}

func TestHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		repo       *fakeRepository
		id         string
		wantStatus int
	}{
		{"เจอใบกำกับ ได้ 200", issuedRepo(), invoiceID, http.StatusOK},
		{"ไม่มีใบนี้ ได้ 404", issuedRepo(), missingID, http.StatusNotFound},
		{"id ไม่ใช่ UUID ได้ 404", issuedRepo(), "INV-2026-0001", http.StatusNotFound},
		{"DB พัง ได้ 500", &fakeRepository{err: errors.New("db down")}, invoiceID, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendRequest(newTestRouter(tt.repo), http.MethodGet, "/api/v1/invoices/"+tt.id, "")
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, ต้องได้ %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestHandler_Create(t *testing.T) {
	rec := sendRequest(newTestRouter(issuedRepo()), http.MethodPost, "/api/v1/invoices", validBody)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, ต้องได้ %d (body: %s)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var got Invoice
	json.NewDecoder(rec.Body).Decode(&got)
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
		{"ไม่มีสินค้าในใบ ได้ 400", `{"issue_date":"2026-10-31","customer_name":"ลูกค้า","items":[]}`, nil, http.StatusBadRequest, `{"error":"invoice_items_required"}`},
		{"ร้านไม่มีเลขผู้เสียภาษี ได้ 409", validBody, ErrSellerTaxIDMissing, http.StatusConflict, `{"error":"company_tax_id_required"}`},
		{"ของในสต็อกไม่พอ ได้ 409", validBody, ErrStockNotEnough, http.StatusConflict, `{"error":"stock_not_enough"}`},
		{"ไม่มีสินค้านี้ ได้ 404", validBody, ErrProductNotFound, http.StatusNotFound, `{"error":"product_not_found"}`},
		{"DB พัง ได้ 500", validBody, errors.New("db down"), http.StatusInternalServerError, `{"error":"server_error"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := issuedRepo()
			repo.createErr = tt.createErr
			rec := sendRequest(newTestRouter(repo), http.MethodPost, "/api/v1/invoices", tt.body)
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
		{"ยกเลิกสำเร็จ ได้ 200 สถานะ cancelled", invoiceID, nil, http.StatusOK, "cancelled"},
		{"ยกเลิกซ้ำ ได้ 409", invoiceID, ErrAlreadyCancelled, http.StatusConflict, ""},
		{"ไม่มีใบนี้ ได้ 404", missingID, ErrNotFound, http.StatusNotFound, ""},
		{"id ไม่ใช่ UUID ได้ 404", "abc", nil, http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := issuedRepo()
			repo.cancelErr = tt.cancelErr
			rec := sendRequest(newTestRouter(repo), http.MethodPost, "/api/v1/invoices/"+tt.id+"/cancel", "")
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, ต้องได้ %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatusText == "" {
				return
			}
			var got Invoice
			json.NewDecoder(rec.Body).Decode(&got)
			if got.Status != tt.wantStatusText {
				t.Errorf("status ของใบ = %q, ต้องได้ %q", got.Status, tt.wantStatusText)
			}
		})
	}
}
