package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/middleware"
	"github.com/naskung/api-Costbook/internal/utils"
)

// fakeOpenAI AI แบบ OpenAI ปลอม: ตอบตามที่เตรียมไว้ทีละอัน + จำคำขอทุกอันไว้ตรวจ
type fakeOpenAI struct {
	replies   []string // คำตอบแบบ JSON ของ /chat/completions
	err       error
	models    []string
	modelsErr error
	sent      []openAIRequest
}

func (f *fakeOpenAI) Complete(ctx context.Context, req openAIRequest) (openAIResponse, error) {
	f.sent = append(f.sent, req)
	if f.err != nil {
		return openAIResponse{}, f.err
	}
	if len(f.sent) > len(f.replies) {
		panic("fakeOpenAI: ไม่ได้เตรียมคำตอบไว้")
	}
	var resp openAIResponse
	if err := json.Unmarshal([]byte(f.replies[len(f.sent)-1]), &resp); err != nil {
		panic(err)
	}
	return resp, nil
}

func (f *fakeOpenAI) Models(ctx context.Context) ([]string, error) {
	return f.models, f.modelsErr
}

// openAIText คำตอบที่เป็นข้อความ
func openAIText(text string) string {
	return `{"choices":[{"message":{"role":"assistant","content":` + jsonString(text) + `},"finish_reason":"stop"}]}`
}

// openAIToolCalls คำตอบที่เรียกเครื่องมือ (calls = JSON array ของ tool_calls)
func openAIToolCalls(calls string) string {
	return `{"choices":[{"message":{"role":"assistant","content":null,"tool_calls":` + calls + `},"finish_reason":"tool_calls"}]}`
}

func openAICall(id, name, arguments string) string {
	return `{"id":"` + id + `","type":"function","function":{"name":"` + name + `","arguments":` + jsonString(arguments) + `}}`
}

// newOpenAIRouter router ที่ร้านตั้ง AI แบบ OpenAI ไว้แล้ว (searcher = nil → ไม่มี Tavily)
func newOpenAIRouter(repo Repository, client *fakeOpenAI, searcher *fakeSearcher) *gin.Engine {
	cfg := Config{GeminiKey: "free", Secret: testSecret}
	if searcher != nil {
		cfg.TavilyKey = "tvly-test"
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(repo, fakeShop{}, cfg)
	h.now = func() time.Time { return testToday }
	h.newGemini = func(apiKey, model string) Gemini { return &fakeGemini{} }
	h.newOpenAI = func(baseURL, apiKey, model string) OpenAI { return client }
	h.newSearcher = func(apiKey string) WebSearcher { return searcher }
	h.RegisterRoutes(r.Group("/api/v1"), middleware.Auth(testSecret))
	return r
}

// deepSeekRepo ร้านที่ตั้ง DeepSeek ไว้ (key เข้ารหัสด้วย testSecret)
func deepSeekRepo(t *testing.T) *fakeRepository {
	encrypted, err := utils.Encrypt(ownDeepSeek.APIKey, testSecret)
	if err != nil {
		t.Fatal(err)
	}
	return &fakeRepository{ai: SavedAI{Provider: providerOpenAI, BaseURL: ownDeepSeek.BaseURL, Model: ownDeepSeek.Model, EncryptedKey: encrypted}}
}

// ---------- ตัวเรียก API (ใช้ server ปลอม) ----------

func TestOpenAIClient_Complete(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody openAIRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &gotBody)
		w.Write([]byte(openAIText("สวัสดีครับ")))
	}))
	defer server.Close()

	client := NewOpenAI(server.URL+"/v1/", "sk-test", "deepseek-chat")
	resp, err := client.Complete(context.Background(), openAIRequest{Messages: []openAIMessage{{Role: "user", Content: "สวัสดี"}}})
	if err != nil || resp.Choices[0].Message.Content != "สวัสดีครับ" {
		t.Fatalf("ได้ %+v %v", resp, err)
	}
	if gotPath != "/v1/chat/completions" || gotAuth != "Bearer sk-test" || gotBody.Model != "deepseek-chat" {
		t.Errorf("ส่งผิด path=%q auth=%q model=%q (ตัด / ท้ายที่อยู่ + ใส่รุ่นให้เอง)", gotPath, gotAuth, gotBody.Model)
	}
}

func TestOpenAIClient_Models_NoKey(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"object":"list","data":[{"id":"llama3.2"},{"id":"qwen3"},{"id":""}]}`))
	}))
	defer server.Close()

	models, err := NewOpenAI(server.URL, "", "").Models(context.Background())
	if err != nil || strings.Join(models, ",") != "llama3.2,qwen3" {
		t.Errorf("ได้ %v %v", models, err)
	}
	if gotAuth != "" {
		t.Errorf("ไม่มี key (เช่น Ollama) ห้ามส่ง Authorization ได้ %q", gotAuth)
	}
}

func TestOpenAIClient_Errors(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr string
	}{
		{"key ผิด แบบ OpenAI", http.StatusUnauthorized, `{"error":{"message":"Incorrect API key"}}`, "openai-compatible 401: Incorrect API key"},
		{"error แบบข้อความตรง ๆ", http.StatusTooManyRequests, `{"error":"rate limited"}`, "openai-compatible 429: rate limited"},
		{"ตอบ 200 แต่ไม่ใช่ JSON (ที่อยู่ผิด เป็นหน้าเว็บ)", http.StatusOK, `<html>hello</html>`, ErrNotOpenAIAPI.Error()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()
			_, err := NewOpenAI(server.URL, "k", "m").Models(context.Background())
			if err == nil || err.Error() != tt.wantErr {
				t.Errorf("ได้ %v ต้องได้ %s", err, tt.wantErr)
			}
		})
	}
}

// ---------- แชท ----------

func TestOpenAIChat_ToolThenProposeThenReply(t *testing.T) {
	client := &fakeOpenAI{replies: []string{
		openAIToolCalls(`[` + openAICall("c1", "list_products", "") + `]`),
		openAIToolCalls(`[` + openAICall("c2", "propose_stock_in", `{"product_id":"`+caseID+`","quantity":10,"unit_cost":50,"supplier":"","note":"","reason":"ผู้ใช้สั่ง"}`) + `]`),
		openAIText("<think>คิดก่อน {x}</think>กดยืนยันด้านล่างได้เลยครับ"),
	}}
	tools := toolbox{shop: fakeShop{}, repo: &fakeRepository{}, today: testToday}
	reply, err := openAIChat(context.Background(), client, tools, []ChatMessage{{Role: "user", Content: "รับเคสเข้า 10 ชิ้น"}})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Reply != "กดยืนยันด้านล่างได้เลยครับ" {
		t.Errorf("ต้องตัดส่วนที่ AI คิดในใจออก ได้ %q", reply.Reply)
	}
	if len(reply.Actions) != 1 || reply.Actions[0].ID != "c2" || reply.Actions[0].Type != actionStockIn {
		t.Errorf("ต้องได้งานรับเข้า 1 งาน ได้ %+v", reply.Actions)
	}

	first := client.sent[0]
	if first.Messages[0].Role != "system" || !strings.Contains(first.Messages[0].Content, "2026-10-31") || len(first.Tools) != len(chatTools) {
		t.Errorf("คำขอแรกต้องมีคำสั่งระบบ + วันนี้ + เครื่องมือครบ ได้ %+v", first.Messages[0])
	}
	second := client.sent[1].Messages
	result := second[len(second)-1]
	if result.Role != "tool" || result.ToolCallID != "c1" || !strings.Contains(result.Content, "เคสโทรศัพท์") {
		t.Errorf("ต้องส่งผล list_products กลับเป็น role tool ได้ %+v", result)
	}
}

func TestOpenAIChat_ToolErrorGoesBackToAI(t *testing.T) {
	client := &fakeOpenAI{replies: []string{
		openAIToolCalls(`[` + openAICall("c1", "propose_stock_out", `{"product_id":"abc","quantity":1,"reason":"x"}`) + `]`),
		openAIText("ขอโทษครับ ไม่เจอสินค้านี้"),
	}}
	tools := toolbox{shop: fakeShop{}, repo: &fakeRepository{}, today: testToday}
	reply, err := openAIChat(context.Background(), client, tools, []ChatMessage{{Role: "user", Content: "ตัดของ"}})
	if err != nil || len(reply.Actions) != 0 {
		t.Fatalf("ข้อมูลผิดต้องส่งกลับให้ AI แก้ ไม่ใช่ error ได้ %+v %v", reply, err)
	}
	sent := client.sent[1].Messages
	if !strings.Contains(sent[len(sent)-1].Content, `"error"`) {
		t.Errorf("ต้องบอก AI ว่าผิด ได้ %+v", sent[len(sent)-1])
	}
}

func TestOpenAIChat_Errors(t *testing.T) {
	tools := toolbox{shop: fakeShop{}, repo: &fakeRepository{}, today: testToday}
	history := []ChatMessage{{Role: "user", Content: "สวัสดี"}}
	tests := []struct {
		name    string
		replies []string
		want    error
	}{
		{"ไม่มีคำตอบ", []string{`{"choices":[]}`}, ErrNoReport},
		{"ถูกกรองเนื้อหา", []string{`{"choices":[{"message":{"role":"assistant","content":""},"finish_reason":"content_filter"}]}`}, ErrRefused},
		{"ตอบว่างเปล่า", []string{openAIText("  ")}, ErrNoReport},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := openAIChat(context.Background(), &fakeOpenAI{replies: tt.replies}, tools, history)
			if !errors.Is(err, tt.want) {
				t.Errorf("ได้ %v ต้องได้ %v", err, tt.want)
			}
		})
	}
}

func TestJSONObjectIn(t *testing.T) {
	tests := map[string]string{
		`{"a":1}`:                         `{"a":1}`,
		"```json\n{\"a\":{\"b\":2}}\n```": `{"a":{"b":2}}`,
		"นี่คือผล: {\"a\":1} หวังว่าช่วยได้": `{"a":1}`,
		"ไม่มี JSON": "",
	}
	for input, want := range tests {
		if got := jsonObjectIn(input); got != want {
			t.Errorf("jsonObjectIn(%q) = %q ต้องได้ %q", input, got, want)
		}
	}
}

// ---------- เช็คราคาตลาด (ค้นเว็บด้วย Tavily) ----------

func TestOpenAIResearch_WithTavily(t *testing.T) {
	queries := openAIText("```json\n" + `{"queries":[{"text":"พวงกุญแจ 3D","source":"marketplace"}]}` + "\n```")
	report := openAIText(`{"summary":"ราคา 49–390 บาท","retail_low":49,"retail_high":390,"retail_typical":99,"wholesale_low":0,"wholesale_high":0,` +
		`"wholesale_note":"","market_cost":0,"cost_items":[],"suggested_price":99,"advice":"","confidence":"medium",` +
		`"sources":[{"title":"Shopee","url":"` + shopeeShop + `","price":49,"kind":"retail"},{"title":"แต่งเอง","url":"https://fake.example","price":1,"kind":"retail"}],"search_suggestions":""}`)
	client := &fakeOpenAI{replies: []string{queries, report}}
	assistant := openAIAssistant{client: client, searcher: &fakeSearcher{results: foundOnWeb}}

	got, err := assistant.Research(context.Background(), "สินค้า: พวงกุญแจพิมพ์ 3D")
	if err != nil {
		t.Fatal(err)
	}
	if got.RetailLow != 49 || len(got.Sources) != 1 || got.Sources[0].URL != shopeeShop {
		t.Errorf("ต้องได้ผล + เก็บเฉพาะลิงก์ที่ค้นเจอจริง ได้ %+v", got)
	}
	if !strings.Contains(client.sent[0].Messages[0].Content, "JSON Schema") {
		t.Error("ต้องบอกรูปแบบ JSON ในคำสั่ง (ไม่ใช้ response_format)")
	}
}

func TestOpenAIResearch_NoTavily(t *testing.T) {
	_, err := openAIAssistant{client: &fakeOpenAI{}}.Research(context.Background(), "สินค้า")
	if !errors.Is(err, ErrNoWebSearch) {
		t.Errorf("ไม่มี Tavily ต้องค้นเว็บไม่ได้ ได้ %v", err)
	}
}

// ---------- handler ----------

func TestHandler_SaveOpenAI(t *testing.T) {
	repo := &fakeRepository{}
	client := &fakeOpenAI{models: []string{"deepseek-chat", "deepseek-reasoner"}}
	body := `{"provider":"openai","base_url":" https://api.deepseek.com/v1/ ","api_key":"` + ownDeepSeek.APIKey + `","model":"deepseek-chat"}`
	rec := sendRequest(newOpenAIRouter(repo, client, nil), http.MethodPut, "/api/v1/ai/key", body)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
	got := rec.Body.String()
	for _, want := range []string{`"provider":"openai"`, `"source":"settings"`, `"model":"deepseek-chat"`, `"base_url":"https://api.deepseek.com/v1"`, `"key_hint":"…Z9y8"`, `"web_search":false`} {
		if !strings.Contains(got, want) {
			t.Errorf("ต้องมี %s ได้ %s", want, got)
		}
	}
	if strings.Contains(got, ownDeepSeek.APIKey) || strings.Contains(repo.ai.EncryptedKey, "sk-deepseek") {
		t.Error("ห้ามส่ง key กลับ และห้ามเก็บ key เป็นข้อความตรง ๆ")
	}
	if repo.ai.BaseURL != "https://api.deepseek.com/v1" || repo.ai.Model != "deepseek-chat" {
		t.Errorf("ต้องเก็บที่อยู่ (ตัดช่องว่าง + / ท้าย) และรุ่น ได้ %+v", repo.ai)
	}
}

func TestHandler_SaveOpenAI_NoKeyAndNoModelList(t *testing.T) {
	repo := &fakeRepository{}
	client := &fakeOpenAI{modelsErr: &OpenAIError{Status: http.StatusNotFound, Message: "not found"}}
	body := `{"provider":"openai","base_url":"http://localhost:11434/v1","model":"llama3.2"}`
	rec := sendRequest(newOpenAIRouter(repo, client, nil), http.MethodPut, "/api/v1/ai/key", body)
	if rec.Code != http.StatusOK || repo.ai.EncryptedKey != "" || repo.ai.Model != "llama3.2" {
		t.Errorf("ไม่มี key (Ollama) + ไม่มีรายชื่อรุ่นให้ ต้องบันทึกได้ ได้ %d %s %+v", rec.Code, rec.Body.String(), repo.ai)
	}
}

func TestHandler_SaveOpenAI_Errors(t *testing.T) {
	valid := `"base_url":"https://api.groq.com/openai/v1","api_key":"gsk-key","model":"llama-3.3-70b"`
	tests := []struct {
		name     string
		client   *fakeOpenAI
		body     string
		wantCode int
		wantBody string
	}{
		{"provider ไม่รู้จัก", &fakeOpenAI{}, `{"provider":"abc"}`, http.StatusBadRequest, "ai_provider_invalid"},
		{"ที่อยู่ไม่ใช่ http(s)", &fakeOpenAI{}, `{"provider":"openai","base_url":"api.groq.com","model":"x"}`, http.StatusBadRequest, "ai_base_url_invalid"},
		{"ไม่ใส่รุ่น", &fakeOpenAI{}, `{"provider":"openai","base_url":"https://api.groq.com/openai/v1","model":" "}`, http.StatusBadRequest, "ai_model_required"},
		{"key มีช่องว่างตรงกลาง", &fakeOpenAI{}, `{"provider":"openai","base_url":"https://x.ai/v1","model":"m","api_key":"a b"}`, http.StatusBadRequest, "ai_key_format_invalid"},
		{"key ผิด", &fakeOpenAI{modelsErr: &OpenAIError{Status: http.StatusUnauthorized}}, `{"provider":"openai",` + valid + `}`, http.StatusBadRequest, "ai_key_invalid"},
		{"ไม่มีรุ่นที่เลือก", &fakeOpenAI{models: []string{"other"}}, `{"provider":"openai",` + valid + `}`, http.StatusBadRequest, "ai_model_not_found"},
		{"ต่อไม่ได้เลย", &fakeOpenAI{modelsErr: errors.New("dial tcp: connection refused")}, `{"provider":"openai",` + valid + `}`, http.StatusBadRequest, "ai_connect_failed"},
		{"ที่อยู่ไม่ใช่ API แบบ OpenAI", &fakeOpenAI{modelsErr: ErrNotOpenAIAPI}, `{"provider":"openai",` + valid + `}`, http.StatusBadRequest, "ai_connect_failed"},
		{"ปลายทางไม่ว่าง", &fakeOpenAI{modelsErr: &OpenAIError{Status: http.StatusBadGateway}}, `{"provider":"openai",` + valid + `}`, http.StatusServiceUnavailable, "ai_busy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepository{}
			rec := sendRequest(newOpenAIRouter(repo, tt.client, nil), http.MethodPut, "/api/v1/ai/key", tt.body)
			if rec.Code != tt.wantCode || rec.Body.String() != `{"error":"`+tt.wantBody+`"}` {
				t.Errorf("ได้ %d %s ต้องได้ %d %s", rec.Code, rec.Body.String(), tt.wantCode, tt.wantBody)
			}
			if repo.ai.Provider != "" {
				t.Error("ใช้ไม่ได้ ต้องไม่บันทึก")
			}
		})
	}
}

func TestHandler_Models(t *testing.T) {
	client := &fakeOpenAI{models: []string{"gpt-b", "gpt-a"}}
	router := newOpenAIRouter(&fakeRepository{}, client, nil)
	rec := sendRequest(router, http.MethodPost, "/api/v1/ai/models", `{"base_url":"https://api.openai.com/v1","api_key":"sk-x"}`)
	if rec.Code != http.StatusOK || rec.Body.String() != `{"models":["gpt-a","gpt-b"]}` {
		t.Errorf("ต้องได้รายชื่อรุ่นเรียงตามตัวอักษร ได้ %d %s", rec.Code, rec.Body.String())
	}
	rec = sendRequest(router, http.MethodPost, "/api/v1/ai/models", `{"base_url":"ftp://x"}`)
	if rec.Code != http.StatusBadRequest || rec.Body.String() != `{"error":"ai_base_url_invalid"}` {
		t.Errorf("ที่อยู่ผิดต้องได้ 400 ได้ %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Chat_OpenAI(t *testing.T) {
	client := &fakeOpenAI{replies: []string{openAIText("ยอดขายเดือนนี้ 1,450 บาท")}}
	rec := sendRequest(newOpenAIRouter(deepSeekRepo(t), client, nil), http.MethodPost, "/api/v1/ai/chat", askBody)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "1,450") {
		t.Errorf("ร้านตั้ง AI แบบ OpenAI แล้วแชทต้องใช้ตัวนั้น ได้ %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Chat_OpenAIErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"key ใช้ไม่ได้แล้ว", &OpenAIError{Status: http.StatusUnauthorized}, http.StatusServiceUnavailable, "ai_key_invalid"},
		{"เครดิตหมด", &OpenAIError{Status: http.StatusPaymentRequired}, http.StatusServiceUnavailable, "ai_credit_low"},
		{"ไม่มีรุ่นนี้แล้ว", &OpenAIError{Status: http.StatusNotFound}, http.StatusServiceUnavailable, "ai_model_not_found"},
		{"โควตาเต็ม", &OpenAIError{Status: http.StatusTooManyRequests}, http.StatusServiceUnavailable, "ai_quota_exceeded"},
		{"ไม่ว่าง", &OpenAIError{Status: http.StatusServiceUnavailable}, http.StatusServiceUnavailable, "ai_busy"},
		{"อื่น ๆ", &OpenAIError{Status: http.StatusBadRequest, Message: "tools not supported"}, http.StatusBadGateway, "ai_error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := sendRequest(newOpenAIRouter(deepSeekRepo(t), &fakeOpenAI{err: tt.err}, nil), http.MethodPost, "/api/v1/ai/chat", askBody)
			if rec.Code != tt.wantCode || rec.Body.String() != `{"error":"`+tt.wantBody+`"}` {
				t.Errorf("ได้ %d %s ต้องได้ %d %s", rec.Code, rec.Body.String(), tt.wantCode, tt.wantBody)
			}
		})
	}
}

func TestHandler_CheckMarket_OpenAI(t *testing.T) {
	rec := sendRequest(newOpenAIRouter(deepSeekRepo(t), &fakeOpenAI{}, nil), http.MethodPost, "/api/v1/products/"+caseID+"/market", "")
	if rec.Code != http.StatusConflict || rec.Body.String() != `{"error":"ai_search_unavailable"}` {
		t.Errorf("ไม่มี Tavily เช็คราคาตลาดไม่ได้ ได้ %d %s", rec.Code, rec.Body.String())
	}

	queries := openAIText(`{"queries":[{"text":"เคสโทรศัพท์","source":"marketplace"}]}`)
	report := openAIText(`{"summary":"s","retail_low":199,"retail_high":390,"retail_typical":250,"wholesale_low":0,"wholesale_high":0,"wholesale_note":"",` +
		`"market_cost":0,"cost_items":[],"suggested_price":250,"advice":"","confidence":"low","sources":[],"search_suggestions":""}`)
	repo := deepSeekRepo(t)
	rec = sendRequest(newOpenAIRouter(repo, &fakeOpenAI{replies: []string{queries, report}}, &fakeSearcher{results: foundOnWeb}),
		http.MethodPost, "/api/v1/products/"+caseID+"/market", "")
	if rec.Code != http.StatusCreated || repo.saved == nil || repo.saved.RetailTypical != 250 {
		t.Errorf("มี Tavily ต้องเช็คได้และบันทึกผล ได้ %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_DeleteOpenAI(t *testing.T) {
	repo := deepSeekRepo(t)
	rec := sendRequest(newOpenAIRouter(repo, &fakeOpenAI{}, nil), http.MethodDelete, "/api/v1/ai/key", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"provider":"gemini"`) || repo.ai.Provider != "" {
		t.Errorf("ลบแล้วต้องกลับไปใช้ Gemini ฟรี ได้ %d %s", rec.Code, rec.Body.String())
	}
}
