package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/features/company"
	"github.com/naskung/api-Costbook/internal/features/cost"
	"github.com/naskung/api-Costbook/internal/features/dashboard"
	"github.com/naskung/api-Costbook/internal/features/field"
	"github.com/naskung/api-Costbook/internal/features/product"
	"github.com/naskung/api-Costbook/internal/features/transaction"
	"github.com/naskung/api-Costbook/internal/middleware"
	"github.com/naskung/api-Costbook/internal/utils"
)

const (
	testSecret = "test-secret"
	caseID     = "11111111-1111-1111-1111-111111111111"
	missingID  = "99999999-9999-9999-9999-999999999999"
)

var testToday = time.Date(2026, 10, 31, 10, 0, 0, 0, time.UTC)

// fakeClaude Claude ปลอม: ตอบตามที่เตรียมไว้ทีละอัน + จำคำขอทุกอันไว้ตรวจ
type fakeClaude struct {
	replies  []string // คำตอบแบบ JSON ของ API จริง
	err      error
	checkErr error // error ตอนลอง key (CheckKey)
	sent     []anthropic.BetaMessageNewParams
}

func (f *fakeClaude) CheckKey(ctx context.Context) error {
	return f.checkErr
}

func (f *fakeClaude) Send(ctx context.Context, params anthropic.BetaMessageNewParams) (*anthropic.BetaMessage, error) {
	f.sent = append(f.sent, params)
	if f.err != nil {
		return nil, f.err
	}
	if len(f.sent) > len(f.replies) {
		panic("fakeClaude: ไม่ได้เตรียมคำตอบไว้")
	}
	var m anthropic.BetaMessage
	if err := json.Unmarshal([]byte(f.replies[len(f.sent)-1]), &m); err != nil {
		panic(err)
	}
	return &m, nil
}

// lastRequestJSON คำขอล่าสุดแบบ JSON (ไว้เช็คว่าส่งผลเครื่องมือกลับไปถูกไหม)
func (f *fakeClaude) lastRequestJSON() string {
	data, _ := json.Marshal(f.sent[len(f.sent)-1].Messages)
	return string(data)
}

// fakeRepository ตาราง market_checks + key ของ AI ปลอม
type fakeRepository struct {
	latest       MarketCheck
	found        bool
	saved        *MarketReport
	encryptedKey string
}

func (f *fakeRepository) SavedClaudeKey(ctx context.Context) (string, error) {
	return f.encryptedKey, nil
}

func (f *fakeRepository) SaveClaudeKey(ctx context.Context, encrypted string) error {
	f.encryptedKey = encrypted
	return nil
}

func (f *fakeRepository) LatestMarketCheck(ctx context.Context, productID string) (MarketCheck, error) {
	if !f.found {
		return MarketCheck{}, ErrNotFound
	}
	return f.latest, nil
}

func (f *fakeRepository) SaveMarketCheck(ctx context.Context, p product.Product, report MarketReport) (MarketCheck, error) {
	f.saved = &report
	return MarketCheck{ID: "m1", ProductID: p.ID, OurCostPrice: p.CostPrice, OurSellPrice: p.SellPrice, MarketReport: report}, nil
}

// fakeShop ข้อมูลร้านปลอม มีสินค้า 1 ตัว (เคสโทรศัพท์ คงเหลือ 8)
type fakeShop struct{}

var phoneCase = product.Product{ID: caseID, Name: "เคสโทรศัพท์", Unit: "ชิ้น", CostPrice: 52.8, SellPrice: 250, Stock: 8, TrackStock: true}

func (fakeShop) Company(ctx context.Context) (company.Company, error) {
	return company.Company{Name: "MAKEHUB", VATRegistered: true, PricesIncludeVAT: true}, nil
}
func (fakeShop) Fields(ctx context.Context) ([]field.ProductField, error) { return nil, nil }
func (fakeShop) Products(ctx context.Context) ([]product.Product, error) {
	return []product.Product{phoneCase}, nil
}
func (fakeShop) Product(ctx context.Context, id string) (product.Product, error) {
	if id != caseID {
		return product.Product{}, product.ErrNotFound
	}
	return phoneCase, nil
}
func (fakeShop) CostSheet(ctx context.Context, productID string) (cost.CostSheet, error) {
	return cost.CostSheet{ProductID: productID}, nil
}
func (fakeShop) Dashboard(ctx context.Context, monthStart string) (dashboard.Dashboard, error) {
	return dashboard.Dashboard{Month: monthStart[:7]}, nil
}
func (fakeShop) Transactions(ctx context.Context, monthStart string) ([]transaction.Transaction, error) {
	return []transaction.Transaction{}, nil
}

// envClaudeKey key ของ Claude ใน .env (ปลอม)
const envClaudeKey = "sk-ant-env-key-0000ENV1"

// newTestRouter router ที่ใช้ Claude ปลอม (claude = nil → ไม่มี AI ให้ใช้เลย)
func newTestRouter(repo Repository, claude Claude) *gin.Engine {
	cfg := Config{Secret: testSecret}
	if claude != nil {
		cfg.ClaudeKey = envClaudeKey
	}
	return newRouterWith(repo, cfg, claude, nil)
}

// newRouterWith router ที่กำหนด config + Claude / Gemini ปลอมเองได้
func newRouterWith(repo Repository, cfg Config, claude Claude, gemini Gemini) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(repo, fakeShop{}, cfg)
	h.now = func() time.Time { return testToday }
	h.newClaude = func(apiKey string) Claude { return claude }
	h.newGemini = func(apiKey, model string) Gemini { return gemini }
	h.RegisterRoutes(r.Group("/api/v1"), middleware.Auth(testSecret))
	return r
}

func sendRequest(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	token, _ := utils.CreateToken("user-1", testSecret, time.Hour)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// nilClaude ไม่มี API key (NewClaude("") คืน nil)
var nilClaude = NewClaude("")

const askBody = `{"messages":[{"role":"user","content":"รับเคสเข้า 10 ชิ้น ทุนชิ้นละ 50"}]}`

// คำตอบของ Claude แบบต่าง ๆ (รูปแบบเดียวกับ API จริง)
const (
	textReply      = `{"id":"m","type":"message","role":"assistant","stop_reason":"end_turn","content":[{"type":"text","text":"ยอดขายเดือนนี้ 1,450 บาท"}]}`
	refusalReply   = `{"id":"m","type":"message","role":"assistant","stop_reason":"refusal","content":[]}`
	listToolReply  = `{"id":"m","type":"message","role":"assistant","stop_reason":"tool_use","content":[{"type":"tool_use","id":"t1","name":"list_products","input":{}}]}`
	stockToolReply = `{"id":"m","type":"message","role":"assistant","stop_reason":"tool_use","content":[{"type":"text","text":"จะรับเข้าให้นะครับ"},{"type":"tool_use","id":"t2","name":"propose_stock_in","input":{"product_id":"` + caseID + `","quantity":10,"unit_cost":50,"supplier":"","note":"","reason":"ผู้ใช้สั่งรับเข้า"}}]}`
	badIDToolReply = `{"id":"m","type":"message","role":"assistant","stop_reason":"tool_use","content":[{"type":"tool_use","id":"t3","name":"propose_stock_out","input":{"product_id":"abc","quantity":1,"reason":"x"}}]}`
	confirmReply   = `{"id":"m","type":"message","role":"assistant","stop_reason":"end_turn","content":[{"type":"text","text":"กดยืนยันด้านล่างได้เลยครับ"}]}`
)

func TestHandler_Status(t *testing.T) {
	rec := sendRequest(newTestRouter(&fakeRepository{}, nilClaude), http.MethodGet, "/api/v1/ai/status", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"enabled":false,"provider":"none"`) {
		t.Errorf("ไม่มี key เลย ต้องได้ enabled false ได้ %d %s", rec.Code, rec.Body.String())
	}
	rec = sendRequest(newTestRouter(&fakeRepository{}, &fakeClaude{}), http.MethodGet, "/api/v1/ai/status", "")
	body := rec.Body.String()
	if !strings.Contains(body, `"provider":"claude"`) || !strings.Contains(body, `"source":"env"`) || !strings.Contains(body, model) {
		t.Errorf("มี key ใน .env ต้องได้ Claude จาก env + ชื่อรุ่น ได้ %s", body)
	}
	if strings.Contains(body, envClaudeKey) {
		t.Error("ห้ามส่ง key ตัวจริงกลับไป")
	}
}

func TestHandler_Chat_TextReply(t *testing.T) {
	claude := &fakeClaude{replies: []string{textReply}}
	rec := sendRequest(newTestRouter(&fakeRepository{}, claude), http.MethodPost, "/api/v1/ai/chat", askBody)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d ต้องได้ 200 (%s)", rec.Code, rec.Body.String())
	}
	var got ChatReply
	json.NewDecoder(rec.Body).Decode(&got)
	if got.Reply != "ยอดขายเดือนนี้ 1,450 บาท" || len(got.Actions) != 0 {
		t.Errorf("ได้ %+v", got)
	}
	system := claude.sent[0].System
	if len(system) != 2 || !strings.Contains(system[1].Text, "2026-10-31") {
		t.Errorf("คำขอต้องมีคำสั่งหลัก + วันที่วันนี้ ได้ %+v", system)
	}
}

func TestHandler_Chat_ReadToolThenReply(t *testing.T) {
	claude := &fakeClaude{replies: []string{listToolReply, textReply}}
	rec := sendRequest(newTestRouter(&fakeRepository{}, claude), http.MethodPost, "/api/v1/ai/chat", askBody)

	if rec.Code != http.StatusOK || len(claude.sent) != 2 {
		t.Fatalf("ต้องส่งคำขอ 2 รอบ แล้วได้ 200 ได้ %d รอบ status %d", len(claude.sent), rec.Code)
	}
	sent := claude.lastRequestJSON()
	if !strings.Contains(sent, `"tool_use_id":"t1"`) || !strings.Contains(sent, "เคสโทรศัพท์") {
		t.Errorf("รอบ 2 ต้องส่งรายการสินค้ากลับไปให้ AI ได้ %s", sent)
	}
}

func TestHandler_Chat_ProposeAction(t *testing.T) {
	claude := &fakeClaude{replies: []string{stockToolReply, confirmReply}}
	rec := sendRequest(newTestRouter(&fakeRepository{}, claude), http.MethodPost, "/api/v1/ai/chat", askBody)

	var got struct {
		Reply   string `json:"reply"`
		Actions []struct {
			ID          string         `json:"id"`
			Type        string         `json:"type"`
			ProductName string         `json:"product_name"`
			Data        map[string]any `json:"data"`
		} `json:"actions"`
	}
	json.NewDecoder(rec.Body).Decode(&got)
	if rec.Code != http.StatusOK || len(got.Actions) != 1 {
		t.Fatalf("ต้องได้งานที่เสนอ 1 งาน ได้ %d %+v", rec.Code, got)
	}
	action := got.Actions[0]
	if action.ID != "t2" || action.Type != actionStockIn || action.ProductName != "เคสโทรศัพท์" {
		t.Errorf("งานที่เสนอผิด ได้ %+v", action)
	}
	if action.Data["quantity"] != 10.0 || action.Data["unit_cost"] != 50.0 || action.Data["product_id"] != caseID {
		t.Errorf("data ต้องเป็นข้อมูลที่ POST /stock/in รับ ได้ %+v", action.Data)
	}
	if !strings.Contains(claude.lastRequestJSON(), proposedText) {
		t.Error("ต้องบอก AI ว่ายังไม่ได้ทำ รอผู้ใช้กดยืนยัน")
	}
}

func TestHandler_Chat_ToolErrorGoesBackToAI(t *testing.T) {
	claude := &fakeClaude{replies: []string{badIDToolReply, textReply}}
	rec := sendRequest(newTestRouter(&fakeRepository{}, claude), http.MethodPost, "/api/v1/ai/chat", askBody)

	if rec.Code != http.StatusOK {
		t.Fatalf("id สินค้าผิด ต้องส่งกลับให้ AI แก้ ไม่ใช่ error ได้ %d", rec.Code)
	}
	sent := claude.lastRequestJSON()
	if !strings.Contains(sent, `"is_error":true`) || !strings.Contains(sent, "ไม่เจอสินค้า") {
		t.Errorf("ต้องส่ง tool_result แบบ is_error ได้ %s", sent)
	}
	if strings.Contains(rec.Body.String(), "stock_out") {
		t.Error("งานที่ข้อมูลผิดต้องไม่อยู่ในรายการที่เสนอ")
	}
}

func TestHandler_Chat_Errors(t *testing.T) {
	tests := []struct {
		name       string
		claude     Claude
		body       string
		wantStatus int
		wantBody   string
	}{
		{"ไม่มี API key ได้ 503", nilClaude, askBody, http.StatusServiceUnavailable, `{"error":"ai_not_configured"}`},
		{"JSON พัง ได้ 400", &fakeClaude{}, `{`, http.StatusBadRequest, `{"error":"invalid_json"}`},
		{"ประวัติแชทผิด ได้ 400", &fakeClaude{}, `{"messages":[]}`, http.StatusBadRequest, `{"error":"ai_messages_invalid"}`},
		{"AI ปฏิเสธ ได้ 422", &fakeClaude{replies: []string{refusalReply}}, askBody, http.StatusUnprocessableEntity, `{"error":"ai_refused"}`},
		{"API key ผิด ได้ 503", &fakeClaude{err: &anthropic.Error{StatusCode: http.StatusUnauthorized}}, askBody, http.StatusServiceUnavailable, `{"error":"ai_key_invalid"}`},
		{"Claude ไม่ว่าง ได้ 503", &fakeClaude{err: &anthropic.Error{StatusCode: http.StatusTooManyRequests}}, askBody, http.StatusServiceUnavailable, `{"error":"ai_busy"}`},
		{"หมดเวลา ได้ 504", &fakeClaude{err: context.DeadlineExceeded}, askBody, http.StatusGatewayTimeout, `{"error":"ai_timeout"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendRequest(newTestRouter(&fakeRepository{}, tt.claude), http.MethodPost, "/api/v1/ai/chat", tt.body)
			if rec.Code != tt.wantStatus || strings.TrimSpace(rec.Body.String()) != tt.wantBody {
				t.Errorf("ได้ %d %s ต้องได้ %d %s", rec.Code, rec.Body.String(), tt.wantStatus, tt.wantBody)
			}
		})
	}
}

func TestHandler_Chat_TooManySteps(t *testing.T) {
	replies := []string{}
	for i := 0; i < maxChatRounds; i++ {
		replies = append(replies, listToolReply)
	}
	rec := sendRequest(newTestRouter(&fakeRepository{}, &fakeClaude{replies: replies}), http.MethodPost, "/api/v1/ai/chat", askBody)
	if rec.Code != http.StatusBadGateway || !strings.Contains(rec.Body.String(), "ai_no_result") {
		t.Errorf("AI เรียกเครื่องมือไม่จบ ต้องหยุดที่ %d รอบ ได้ %d %s", maxChatRounds, rec.Code, rec.Body.String())
	}
}

// ---------- เช็คราคาตลาด ----------

const (
	searchPauseReply = `{"id":"m","type":"message","role":"assistant","stop_reason":"pause_turn","content":[{"type":"server_tool_use","id":"s1","name":"web_search","input":{"query":"เคสโทรศัพท์ 3D ราคา"}}]}`
	reportReply      = `{"id":"m","type":"message","role":"assistant","stop_reason":"tool_use","content":[{"type":"tool_use","id":"r1","name":"save_market_report","input":{
		"summary":"ราคาส่วนใหญ่ 199–390 บาท","retail_low":199,"retail_high":390,"retail_typical":259,
		"wholesale_low":120,"wholesale_high":150,"wholesale_note":"ขั้นต่ำ 50 ชิ้น","market_cost":45,
		"cost_items":[{"name":"เส้น PLA","amount":18,"note":"40 กรัม × 0.45"}],
		"suggested_price":259,"advice":"ตั้ง 259 บาท","confidence":"medium",
		"sources":[{"title":"Shopee","url":"https://shopee.co.th/x","price":259,"kind":"retail"}]}}]}`
	forgotReportReply = `{"id":"m","type":"message","role":"assistant","stop_reason":"end_turn","content":[{"type":"text","text":"ราคาประมาณ 250 บาท"}]}`
)

func TestHandler_CheckMarket(t *testing.T) {
	repo := &fakeRepository{}
	claude := &fakeClaude{replies: []string{searchPauseReply, reportReply}}
	rec := sendRequest(newTestRouter(repo, claude), http.MethodPost, "/api/v1/products/"+caseID+"/market", "")

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d ต้องได้ 201 (%s)", rec.Code, rec.Body.String())
	}
	if len(claude.sent) != 2 {
		t.Errorf("pause_turn ต้องส่งต่อให้ค้นต่อ 1 รอบ ได้ %d รอบ", len(claude.sent))
	}
	if repo.saved == nil || repo.saved.RetailTypical != 259 || repo.saved.Sources[0].URL != "https://shopee.co.th/x" {
		t.Errorf("ต้องบันทึกผลที่ AI ส่งมา ได้ %+v", repo.saved)
	}
	firstPrompt, _ := json.Marshal(claude.sent[0].Messages)
	if !strings.Contains(string(firstPrompt), "เคสโทรศัพท์") || !strings.Contains(string(firstPrompt), "รวม VAT แล้ว") {
		t.Errorf("ต้องส่งข้อมูลสินค้า + รูปแบบ VAT ให้ AI ได้ %s", firstPrompt)
	}
	var got MarketCheck
	json.NewDecoder(rec.Body).Decode(&got)
	if got.OurSellPrice != 250 || got.Confidence != "medium" {
		t.Errorf("ผลที่ตอบกลับต้องมีราคาของร้าน + ผลจาก AI ได้ %+v", got)
	}
}

func TestHandler_CheckMarket_RemindsOnceThenFails(t *testing.T) {
	claude := &fakeClaude{replies: []string{forgotReportReply, forgotReportReply}}
	rec := sendRequest(newTestRouter(&fakeRepository{}, claude), http.MethodPost, "/api/v1/products/"+caseID+"/market", "")

	if rec.Code != http.StatusBadGateway || !strings.Contains(rec.Body.String(), "ai_no_result") {
		t.Errorf("ไม่ส่งผล 2 รอบ ต้องได้ 502 ai_no_result ได้ %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(claude.lastRequestJSON(), remindReportText) {
		t.Error("รอบที่ 2 ต้องเตือนให้เรียก save_market_report")
	}
}

func TestHandler_CheckMarket_RemindThenReport(t *testing.T) {
	repo := &fakeRepository{}
	claude := &fakeClaude{replies: []string{forgotReportReply, reportReply}}
	rec := sendRequest(newTestRouter(repo, claude), http.MethodPost, "/api/v1/products/"+caseID+"/market", "")
	if rec.Code != http.StatusCreated || repo.saved == nil {
		t.Errorf("เตือนแล้วส่งผล ต้องได้ 201 ได้ %d", rec.Code)
	}
}

func TestHandler_CheckMarket_Errors(t *testing.T) {
	tests := []struct {
		name       string
		claude     Claude
		path       string
		wantStatus int
		wantBody   string
	}{
		{"ไม่มี API key ได้ 503", nilClaude, "/api/v1/products/" + caseID + "/market", http.StatusServiceUnavailable, `{"error":"ai_not_configured"}`},
		{"ไม่มีสินค้านี้ ได้ 404", &fakeClaude{}, "/api/v1/products/" + missingID + "/market", http.StatusNotFound, `{"error":"product_not_found"}`},
		{"id ไม่ใช่ UUID ได้ 404", &fakeClaude{}, "/api/v1/products/abc/market", http.StatusNotFound, `{"error":"product_not_found"}`},
		{"AI ปฏิเสธ ได้ 422", &fakeClaude{replies: []string{refusalReply}}, "/api/v1/products/" + caseID + "/market", http.StatusUnprocessableEntity, `{"error":"ai_refused"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendRequest(newTestRouter(&fakeRepository{}, tt.claude), http.MethodPost, tt.path, "")
			if rec.Code != tt.wantStatus || strings.TrimSpace(rec.Body.String()) != tt.wantBody {
				t.Errorf("ได้ %d %s ต้องได้ %d %s", rec.Code, rec.Body.String(), tt.wantStatus, tt.wantBody)
			}
		})
	}
}

func TestHandler_GetMarketCheck(t *testing.T) {
	rec := sendRequest(newTestRouter(&fakeRepository{}, nilClaude), http.MethodGet, "/api/v1/products/"+caseID+"/market", "")
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "market_check_not_found") {
		t.Errorf("ยังไม่เคยเช็ค ต้องได้ 404 ได้ %d %s", rec.Code, rec.Body.String())
	}

	repo := &fakeRepository{found: true, latest: MarketCheck{ID: "m1", ProductID: caseID, MarketReport: MarketReport{RetailTypical: 259}}}
	rec = sendRequest(newTestRouter(repo, nilClaude), http.MethodGet, "/api/v1/products/"+caseID+"/market", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"retail_typical":259`) {
		t.Errorf("ดูผลเดิมได้แม้ไม่มี API key ได้ %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_RequiresLogin(t *testing.T) {
	router := newTestRouter(&fakeRepository{}, &fakeClaude{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", bytes.NewBufferString(askBody))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("ไม่ได้ login ต้องได้ 401 ได้ %d", rec.Code)
	}
}
