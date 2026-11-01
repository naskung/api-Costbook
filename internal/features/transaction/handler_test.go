package transaction

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
	expenseID   = "11111111-1111-1111-1111-111111111111"
	fromInvoice = "22222222-2222-2222-2222-222222222222"
	missingID   = "99999999-9999-9999-9999-999999999999"
	validBody   = `{"type":"expense","category":"material","amount":250.5,"date":"2026-10-31","note":"ซื้อแป้ง"}`
)

// fakeRepository ฐานข้อมูลปลอม ใช้แทน Postgres ตอน test
type fakeRepository struct {
	transactions  []Transaction
	gotMonthStart string
	amounts       MonthAmounts
	err           error // ใส่ค่านี้เมื่ออยากจำลองว่า DB พัง
}

func (f *fakeRepository) List(ctx context.Context, monthStart string) ([]Transaction, error) {
	f.gotMonthStart = monthStart
	return f.transactions, f.err
}

func (f *fakeRepository) MonthAmounts(ctx context.Context, monthStart string) (MonthAmounts, error) {
	f.gotMonthStart = monthStart
	return f.amounts, f.err
}

func (f *fakeRepository) Create(ctx context.Context, req TransactionRequest) (Transaction, error) {
	if f.err != nil {
		return Transaction{}, f.err
	}
	return Transaction{ID: "new-id", Type: req.Type, Category: req.Category, Amount: req.Amount, Date: req.Date, Note: req.Note}, nil
}

func (f *fakeRepository) Update(ctx context.Context, id string, req TransactionRequest) (Transaction, error) {
	err := f.check(id)
	if err != nil {
		return Transaction{}, err
	}
	return Transaction{ID: id, Type: req.Type, Amount: req.Amount}, nil
}

func (f *fakeRepository) Delete(ctx context.Context, id string) error {
	return f.check(id)
}

// check จำลองกฎของ DB: รายการจากใบกำกับแก้ / ลบไม่ได้
func (f *fakeRepository) check(id string) error {
	if f.err != nil {
		return f.err
	}
	if id == fromInvoice {
		return ErrFromInvoice
	}
	if id != expenseID {
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

func TestHandler_List(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		wantStatus     int
		wantMonthStart string
	}{
		{"month=2026-10 ได้รายการของเดือนนั้น", "/api/v1/transactions?month=2026-10", http.StatusOK, "2026-10-01"},
		{"month ผิดรูปแบบ ได้ 400", "/api/v1/transactions?month=abc", http.StatusBadRequest, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepository{transactions: []Transaction{}}
			rec := sendRequest(newTestRouter(repo), http.MethodGet, tt.path, "")
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, ต้องได้ %d", rec.Code, tt.wantStatus)
			}
			if repo.gotMonthStart != tt.wantMonthStart {
				t.Errorf("monthStart = %q, ต้องได้ %q", repo.gotMonthStart, tt.wantMonthStart)
			}
		})
	}
}

func TestHandler_List_RequiresLogin(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter(&fakeRepository{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/transactions", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, ต้องได้ %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHandler_Summary(t *testing.T) {
	repo := &fakeRepository{amounts: MonthAmounts{Income: 1070, Expense: 250.5, Sales: 1000, CostOfSales: 600}}
	rec := sendRequest(newTestRouter(repo), http.MethodGet, "/api/v1/transactions/summary?month=2026-10", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ต้องได้ %d", rec.Code, http.StatusOK)
	}
	var got Summary
	json.NewDecoder(rec.Body).Decode(&got)
	want := NewSummary("2026-10", repo.amounts)
	if want.Profit != 149.5 {
		t.Fatalf("กำไรสุทธิที่คาดไว้ = %v, ต้องได้ 1,000 − 600 − 250.5 = 149.5", want.Profit)
	}
	if got != want {
		t.Errorf("ได้ %+v, ต้องได้ %+v", got, want)
	}
}

func TestHandler_Summary_DBError(t *testing.T) {
	rec := sendRequest(newTestRouter(&fakeRepository{err: errors.New("db down")}), http.MethodGet, "/api/v1/transactions/summary", "")

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
		{"ข้อมูลถูก ได้ 201", validBody, http.StatusCreated, ""},
		{"JSON ผิดรูปแบบ ได้ 400", `{`, http.StatusBadRequest, `{"error":"invalid_json"}`},
		{"จำนวนเงิน 0 ได้ 400", `{"type":"income","category":"sales","amount":0,"date":"2026-10-31"}`, http.StatusBadRequest, `{"error":"transaction_amount_invalid"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendRequest(newTestRouter(&fakeRepository{}), http.MethodPost, "/api/v1/transactions", tt.body)
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
		wantBody   string
	}{
		{"แก้สำเร็จ ได้ 200", http.MethodPut, expenseID, http.StatusOK, ""},
		{"แก้รายการจากใบกำกับ ได้ 409", http.MethodPut, fromInvoice, http.StatusConflict, `{"error":"transaction_from_invoice"}`},
		{"แก้รายการที่ไม่มี ได้ 404", http.MethodPut, missingID, http.StatusNotFound, `{"error":"transaction_not_found"}`},
		{"ลบสำเร็จ ได้ 204", http.MethodDelete, expenseID, http.StatusNoContent, ""},
		{"ลบรายการจากใบกำกับ ได้ 409", http.MethodDelete, fromInvoice, http.StatusConflict, `{"error":"transaction_from_invoice"}`},
		{"ลบ id ที่ไม่ใช่ UUID ได้ 404", http.MethodDelete, "abc", http.StatusNotFound, `{"error":"transaction_not_found"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendRequest(newTestRouter(&fakeRepository{}), tt.method, "/api/v1/transactions/"+tt.id, validBody)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, ต้องได้ %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantBody != "" && rec.Body.String() != tt.wantBody {
				t.Errorf("body = %s, ต้องได้ %s", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestHandler_Delete_DBError(t *testing.T) {
	rec := sendRequest(newTestRouter(&fakeRepository{err: errors.New("db down")}), http.MethodDelete, "/api/v1/transactions/"+expenseID, "")

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, ต้องได้ %d", rec.Code, http.StatusInternalServerError)
	}
}
