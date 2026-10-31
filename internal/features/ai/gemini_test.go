package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// fakeGemini Gemini ปลอม: ตอบตามที่เตรียมไว้ทีละอัน (JSON แบบ API จริง) + จำคำขอไว้ตรวจ
type fakeGemini struct {
	replies []string
	err     error
	sent    []geminiRequest
}

func (f *fakeGemini) Generate(ctx context.Context, req geminiRequest) (geminiResponse, error) {
	f.sent = append(f.sent, req)
	if f.err != nil {
		return geminiResponse{}, f.err
	}
	if len(f.sent) > len(f.replies) {
		panic("fakeGemini: ไม่ได้เตรียมคำตอบไว้")
	}
	var resp geminiResponse
	if err := json.Unmarshal([]byte(f.replies[len(f.sent)-1]), &resp); err != nil {
		panic(err)
	}
	return resp, nil
}

func (f *fakeGemini) lastRequestJSON() string {
	data, _ := json.Marshal(f.sent[len(f.sent)-1].Contents)
	return string(data)
}

// newGeminiRouter ร้านไม่มี key Claude → ใช้ Gemini ฟรี (ตัวปลอม)
func newGeminiRouter(repo Repository, gemini *fakeGemini) *gin.Engine {
	return newRouterWith(repo, Config{GeminiKey: "free-key", Secret: testSecret}, nil, gemini)
}

// คำตอบของ Gemini แบบต่าง ๆ (รูปแบบเดียวกับ API จริง)
const (
	geminiTextReply    = `{"candidates":[{"content":{"role":"model","parts":[{"text":"ยอดขายเดือนนี้ 1,450 บาท"}]},"finishReason":"STOP"}]}`
	geminiListReply    = `{"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"name":"list_products","args":{}},"thoughtSignature":"SIG123"}]},"finishReason":"STOP"}]}`
	geminiStockReply   = `{"candidates":[{"content":{"role":"model","parts":[{"text":"จะรับเข้าให้นะครับ"},{"functionCall":{"name":"propose_stock_in","args":{"product_id":"` + caseID + `","quantity":10,"unit_cost":50,"reason":"ผู้ใช้สั่ง"}}}]},"finishReason":"STOP"}]}`
	geminiMarketReply  = `{"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"id":"call-9","name":"propose_market_check","args":{"product_id":"` + caseID + `","reason":"x"}}}]},"finishReason":"STOP"}]}`
	geminiBlockedReply = `{"promptFeedback":{"blockReason":"SAFETY"},"candidates":[]}`
	geminiThoughtReply = `{"candidates":[{"content":{"role":"model","parts":[{"text":"กำลังคิด...","thought":true},{"text":"สวัสดีครับ"}]},"finishReason":"STOP"}]}`
	geminiConfirmReply = `{"candidates":[{"content":{"role":"model","parts":[{"text":"กดยืนยันด้านล่างได้เลยครับ"}]},"finishReason":"STOP"}]}`
)

func TestGemini_ChatText(t *testing.T) {
	gemini := &fakeGemini{replies: []string{geminiTextReply}}
	rec := sendRequest(newGeminiRouter(&fakeRepository{}, gemini), http.MethodPost, "/api/v1/ai/chat", askBody)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "ยอดขายเดือนนี้ 1,450 บาท") {
		t.Fatalf("ได้ %d %s", rec.Code, rec.Body.String())
	}
	req := gemini.sent[0]
	system, _ := json.Marshal(req.SystemInstruction)
	if !strings.Contains(string(system), "ผู้ช่วย CostBook") || !strings.Contains(string(system), "2026-10-31") {
		t.Errorf("ต้องส่งคำสั่งหลัก + วันที่วันนี้ ได้ %s", system)
	}
	tools, _ := json.Marshal(req.Tools)
	if !strings.Contains(string(tools), `"functionDeclarations"`) || !strings.Contains(string(tools), "propose_stock_in") {
		t.Errorf("ต้องส่งเครื่องมือแบบ functionDeclarations ได้ %s", tools)
	}
	if strings.Contains(string(tools), `"name":"get_shop_info","parameters"`) {
		t.Error("เครื่องมือที่ไม่มีช่องกรอก ต้องไม่ส่ง parameters")
	}
}

func TestGemini_ToolThenReply_KeepsThoughtSignature(t *testing.T) {
	gemini := &fakeGemini{replies: []string{geminiListReply, geminiTextReply}}
	rec := sendRequest(newGeminiRouter(&fakeRepository{}, gemini), http.MethodPost, "/api/v1/ai/chat", askBody)

	if rec.Code != http.StatusOK || len(gemini.sent) != 2 {
		t.Fatalf("ต้องส่ง 2 รอบ ได้ %d รอบ status %d", len(gemini.sent), rec.Code)
	}
	sent := gemini.lastRequestJSON()
	if !strings.Contains(sent, `"thoughtSignature":"SIG123"`) {
		t.Errorf("ต้องส่งคำตอบเดิมของ Gemini กลับไปครบ (มี thoughtSignature) ได้ %s", sent)
	}
	if !strings.Contains(sent, `"functionResponse"`) || !strings.Contains(sent, "เคสโทรศัพท์") {
		t.Errorf("ต้องส่งผลของ list_products กลับไป ได้ %s", sent)
	}
}

func TestGemini_ProposeAction(t *testing.T) {
	gemini := &fakeGemini{replies: []string{geminiStockReply, geminiConfirmReply}}
	rec := sendRequest(newGeminiRouter(&fakeRepository{}, gemini), http.MethodPost, "/api/v1/ai/chat", askBody)

	var got ChatReply
	json.NewDecoder(rec.Body).Decode(&got)
	if rec.Code != http.StatusOK || len(got.Actions) != 1 {
		t.Fatalf("ต้องได้งานที่เสนอ 1 งาน ได้ %d %+v", rec.Code, got)
	}
	if got.Actions[0].Type != actionStockIn || got.Actions[0].ID != "propose_stock_in_1" {
		t.Errorf("Gemini ไม่ส่ง id มา ต้องตั้ง id ให้เอง ได้ %+v", got.Actions[0])
	}
	if !strings.Contains(gemini.lastRequestJSON(), proposedText) {
		t.Error("ต้องบอก AI ว่ายังไม่ได้ทำ รอผู้ใช้กดยืนยัน")
	}
}

func TestGemini_MarketCheckProposalNeedsClaude(t *testing.T) {
	gemini := &fakeGemini{replies: []string{geminiMarketReply, geminiTextReply}}
	rec := sendRequest(newGeminiRouter(&fakeRepository{}, gemini), http.MethodPost, "/api/v1/ai/chat", askBody)

	sent := gemini.lastRequestJSON()
	if !strings.Contains(sent, `"error":"AI ฟรีค้นเว็บไม่ได้`) || !strings.Contains(sent, `"id":"call-9"`) {
		t.Errorf("AI ฟรีเสนอเช็คราคาตลาดไม่ได้ ต้องบอก AI (พร้อม id เดิม) ได้ %s", sent)
	}
	if strings.Contains(rec.Body.String(), "market_check") {
		t.Error("ต้องไม่มีงานเช็คราคาตลาดในคำตอบ")
	}
}

func TestGemini_SkipsThoughtText(t *testing.T) {
	gemini := &fakeGemini{replies: []string{geminiThoughtReply}}
	rec := sendRequest(newGeminiRouter(&fakeRepository{}, gemini), http.MethodPost, "/api/v1/ai/chat", askBody)
	if strings.Contains(rec.Body.String(), "กำลังคิด") || !strings.Contains(rec.Body.String(), "สวัสดีครับ") {
		t.Errorf("ข้อความที่เป็นความคิดของ AI ต้องไม่แสดง ได้ %s", rec.Body.String())
	}
}

func TestGemini_Errors(t *testing.T) {
	tests := []struct {
		name       string
		gemini     *fakeGemini
		wantStatus int
		wantBody   string
	}{
		{"ไม่ยอมตอบเพราะนโยบาย ได้ 422", &fakeGemini{replies: []string{geminiBlockedReply}}, http.StatusUnprocessableEntity, `{"error":"ai_refused"}`},
		{"โควตาฟรีหมด ได้ 503", &fakeGemini{err: &GeminiError{Status: http.StatusTooManyRequests, Message: "quota"}}, http.StatusServiceUnavailable, `{"error":"ai_quota_exceeded"}`},
		{"key ของ AI ฟรีผิด ได้ 503", &fakeGemini{err: &GeminiError{Status: http.StatusBadRequest, Message: "API key not valid. Please pass a valid API key."}}, http.StatusServiceUnavailable, `{"error":"ai_key_invalid"}`},
		{"Gemini ล่ม ได้ 503", &fakeGemini{err: &GeminiError{Status: http.StatusServiceUnavailable}}, http.StatusServiceUnavailable, `{"error":"ai_busy"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendRequest(newGeminiRouter(&fakeRepository{}, tt.gemini), http.MethodPost, "/api/v1/ai/chat", askBody)
			if rec.Code != tt.wantStatus || strings.TrimSpace(rec.Body.String()) != tt.wantBody {
				t.Errorf("ได้ %d %s ต้องได้ %d %s", rec.Code, rec.Body.String(), tt.wantStatus, tt.wantBody)
			}
		})
	}
}

func TestGemini_MarketCheckNotAvailable(t *testing.T) {
	rec := sendRequest(newGeminiRouter(&fakeRepository{}, &fakeGemini{}), http.MethodPost, "/api/v1/products/"+caseID+"/market", "")
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "ai_search_unavailable") {
		t.Errorf("AI ฟรีเช็คราคาตลาดไม่ได้ ต้องได้ 409 ai_search_unavailable ได้ %d %s", rec.Code, rec.Body.String())
	}
}

func TestGemini_StatusFree(t *testing.T) {
	rec := sendRequest(newGeminiRouter(&fakeRepository{}, &fakeGemini{}), http.MethodGet, "/api/v1/ai/status", "")
	body := rec.Body.String()
	if !strings.Contains(body, `"provider":"gemini"`) || !strings.Contains(body, `"web_search":false`) || !strings.Contains(body, `"free_available":true`) {
		t.Errorf("ได้ %s", body)
	}
}

func TestGeminiClient_Generate(t *testing.T) {
	var gotPath, gotKey, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotKey = r.URL.Path, r.Header.Get("x-goog-api-key")
		data, _ := io.ReadAll(r.Body)
		gotBody = string(data)
		w.Write([]byte(geminiTextReply))
	}))
	defer server.Close()

	client := NewGemini("free-key", "").(*geminiClient)
	client.baseURL = server.URL
	resp, err := client.Generate(context.Background(), geminiRequest{Contents: geminiHistory([]ChatMessage{{Role: "user", Content: "สวัสดี"}})})
	if err != nil || len(resp.Candidates) != 1 {
		t.Fatalf("ได้ %+v %v", resp, err)
	}
	if gotPath != "/models/"+DefaultGeminiModel+":generateContent" || gotKey != "free-key" {
		t.Errorf("path / key ผิด ได้ %q %q", gotPath, gotKey)
	}
	if !strings.Contains(gotBody, `"role":"user"`) || !strings.Contains(gotBody, "สวัสดี") {
		t.Errorf("body ผิด ได้ %s", gotBody)
	}
}

func TestGeminiClient_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"code":429,"message":"Resource has been exhausted","status":"RESOURCE_EXHAUSTED"}}`))
	}))
	defer server.Close()

	client := NewGemini("free-key", "gemini-3.5-flash-lite").(*geminiClient)
	client.baseURL = server.URL
	_, err := client.Generate(context.Background(), geminiRequest{})
	geminiErr, ok := err.(*GeminiError)
	if !ok || geminiErr.Status != http.StatusTooManyRequests || geminiErr.Message != "Resource has been exhausted" {
		t.Errorf("ต้องได้ GeminiError 429 พร้อมข้อความ ได้ %v", err)
	}
}

func TestNewGemini_NoKey(t *testing.T) {
	if NewGemini("", "") != nil {
		t.Error("ไม่มี key ต้องได้ nil (ไม่มี AI ฟรี)")
	}
}
