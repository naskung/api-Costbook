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
	"github.com/naskung/api-Costbook/internal/features/stock"
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
// createErr / cancelErr / deliveryErr ใช้จำลอง error ของ DB เช่น ของไม่พอ ร้านไม่มีเลขภาษี ใบยกเลิกไปแล้ว
type fakeRepository struct {
	invoice       Invoice
	gotMonthStart string
	gotPage       [2]int
	createErr     error
	cancelErr     error
	deliveryErr   error
	err           error
}

func (f *fakeRepository) List(ctx context.Context, monthStart string, limit, offset int) ([]InvoiceSummary, error) {
	f.gotMonthStart = monthStart
	f.gotPage = [2]int{limit, offset}
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
	m := documentAmounts(req.Items, req.Discount, docTaxInvoice, false)
	return Invoice{
		ID: "new-id", DocType: docTaxInvoice, InvoiceNo: invoiceNumber("INV", 2026, 1), Status: "issued",
		CustomerName: req.CustomerName, ItemsTotal: m.itemsTotal, Discount: m.discount,
		Subtotal: m.subtotal, VAT: m.vat, Total: m.total,
	}, nil
}

func (f *fakeRepository) Cancel(ctx context.Context, id string) (Invoice, error) {
	if f.cancelErr != nil {
		return Invoice{}, f.cancelErr
	}
	inv, err := f.FindByID(ctx, id)
	inv.Status = "cancelled"
	return inv, err
}

func (f *fakeRepository) UpdateDelivery(ctx context.Context, id, deliveryStatus string) (Invoice, error) {
	if f.deliveryErr != nil {
		return Invoice{}, f.deliveryErr
	}
	inv, err := f.FindByID(ctx, id)
	inv.DeliveryStatus = deliveryStatus
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

func TestHandler_List_Page(t *testing.T) {
	repo := issuedRepo()
	sendRequest(newTestRouter(repo), http.MethodGet, "/api/v1/invoices?month=2026-10&limit=20&offset=40", "")
	if repo.gotPage != [2]int{20, 40} {
		t.Errorf("ต้องส่ง limit / offset ต่อให้ repository ได้ %v", repo.gotPage)
	}
	rec := sendRequest(newTestRouter(repo), http.MethodGet, "/api/v1/invoices?limit=abc", "")
	if rec.Code != http.StatusBadRequest || rec.Body.String() != `{"error":"page_invalid"}` {
		t.Errorf("limit ผิดต้องได้ 400 page_invalid ได้ %d %s", rec.Code, rec.Body.String())
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
		{"สินค้าเป็นวัสดุ (ไม่ขาย) ได้ 409", validBody, ErrNotForSale, http.StatusConflict, `{"error":"product_not_for_sale"}`},
		{"วัสดุไม่พอทำงานรับพิมพ์ ได้ 409 พร้อมชื่อวัสดุ", validBody, &stock.MaterialShortage{Name: "เส้น PLA", Unit: "กรัม", Need: 60, Have: 20}, http.StatusConflict, `{"error":"material_not_enough","have":20,"material":"เส้น PLA","need":60,"unit":"กรัม"}`},
		{"ไม่มีสินค้านี้ ได้ 404", validBody, ErrProductNotFound, http.StatusNotFound, `{"error":"product_not_found"}`},
		{"ใบกำกับไม่มีชื่อลูกค้า ได้ 400", validBody, ErrCustomerRequired, http.StatusBadRequest, `{"error":"invoice_customer_required"}`},
		{"ส่วนลดเกินยอด ได้ 400", `{"issue_date":"2026-10-31","discount":500,"items":[{"product_id":"` + coffeeID + `","quantity":1,"unit_price":50}]}`, nil, http.StatusBadRequest, `{"error":"invoice_discount_invalid"}`},
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

func TestHandler_UpdateDelivery(t *testing.T) {
	tests := []struct {
		name         string
		id           string
		body         string
		deliveryErr  error
		wantStatus   int
		wantResponse string
	}{
		{"เปลี่ยนเป็นส่งแล้ว ได้ 200", invoiceID, `{"delivery_status":"shipped"}`, nil, http.StatusOK, ""},
		{"สถานะที่ไม่มี ได้ 400", invoiceID, `{"delivery_status":"lost"}`, nil, http.StatusBadRequest, `{"error":"invoice_delivery_status_invalid"}`},
		{"ไม่ส่งสถานะมา ได้ 400", invoiceID, `{}`, nil, http.StatusBadRequest, `{"error":"invoice_delivery_status_invalid"}`},
		{"JSON ผิด ได้ 400", invoiceID, `{`, nil, http.StatusBadRequest, `{"error":"invalid_json"}`},
		{"ใบยกเลิกแล้ว ได้ 409", invoiceID, `{"delivery_status":"received"}`, ErrAlreadyCancelled, http.StatusConflict, `{"error":"invoice_already_cancelled"}`},
		{"ไม่มีใบนี้ ได้ 404", missingID, `{"delivery_status":"received"}`, nil, http.StatusNotFound, `{"error":"invoice_not_found"}`},
		{"id ไม่ใช่ UUID ได้ 404", "abc", `{"delivery_status":"received"}`, nil, http.StatusNotFound, `{"error":"invoice_not_found"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := issuedRepo()
			repo.deliveryErr = tt.deliveryErr
			rec := sendRequest(newTestRouter(repo), http.MethodPut, "/api/v1/invoices/"+tt.id+"/delivery", tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, ต้องได้ %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantResponse != "" && rec.Body.String() != tt.wantResponse {
				t.Errorf("body = %s, ต้องได้ %s", rec.Body.String(), tt.wantResponse)
			}
		})
	}
}

func TestHandler_UpdateDelivery_ReturnsNewStatus(t *testing.T) {
	rec := sendRequest(newTestRouter(issuedRepo()), http.MethodPut, "/api/v1/invoices/"+invoiceID+"/delivery", `{"delivery_status":"shipped"}`)

	var got Invoice
	json.NewDecoder(rec.Body).Decode(&got)
	if got.DeliveryStatus != deliveryShipped {
		t.Errorf("delivery_status = %q, ต้องได้ %q", got.DeliveryStatus, deliveryShipped)
	}
}

func TestHandler_Create_WithDiscount(t *testing.T) {
	body := `{"issue_date":"2026-10-31","customer_name":"ลูกค้า","discount":10,"items":[{"product_id":"` + coffeeID + `","quantity":2,"unit_price":50}]}`
	rec := sendRequest(newTestRouter(issuedRepo()), http.MethodPost, "/api/v1/invoices", body)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, ต้องได้ %d (body: %s)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var got Invoice
	json.NewDecoder(rec.Body).Decode(&got)
	if got.ItemsTotal != 100 || got.Discount != 10 || got.Subtotal != 90 || got.VAT != 6.3 || got.Total != 96.3 {
		t.Errorf("ได้ %+v, ต้องได้ รวม 100 ส่วนลด 10 ก่อน VAT 90 VAT 6.30 สุทธิ 96.30", got)
	}
}
