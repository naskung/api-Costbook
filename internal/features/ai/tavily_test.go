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
)

// fakeSearcher ตัวค้นเว็บปลอม: จำการค้นที่ได้ + ตอบผลเดิมทุกครั้ง (หรือ error)
type fakeSearcher struct {
	results []SearchResult
	err     error
	queries []SearchQuery
}

func (f *fakeSearcher) Search(ctx context.Context, query SearchQuery) ([]SearchResult, error) {
	f.queries = append(f.queries, query)
	return f.results, f.err
}

// searched คำค้น@เว็บ ของทุกครั้งที่ค้น ไว้เทียบใน test (ว่าง = ทั้งเว็บไทย, -เว็บ = ไม่เอาเว็บนั้น)
func (f *fakeSearcher) searched() string {
	parts := []string{}
	for _, q := range f.queries {
		part := q.Text + "@" + strings.Join(q.Sites, "+")
		if len(q.ExcludeSites) > 0 {
			part += "-" + strings.Join(q.ExcludeSites, "-")
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, " | ")
}

const shopeeShop = "https://shopee.co.th/keychain-3d-i.1.2"

var foundOnWeb = []SearchResult{
	{Title: "พวงกุญแจ 3D | Shopee Thailand", URL: shopeeShop, Content: "พวงกุญแจพิมพ์ 3D ราคา ฿49 ส่งฟรี"},
	{Title: "พวงกุญแจ 3D ขายส่ง", URL: "https://example-wholesale.co.th/keychain", Content: "แพ็ก 10 ชิ้น 390 บาท"},
}

// geminiQueriesReply Gemini คิดคำค้น + แหล่ง (ซ้ำ 1 คำ + ว่าง 1 คำ เพื่อทดสอบการกรอง)
var geminiQueriesReply = geminiQueries(`[{"text":"พวงกุญแจ 3D","source":"marketplace"},{"text":"พวงกุญแจ 3D","source":"marketplace"},` +
	`{"text":" ","source":"marketplace"},{"text":"เส้น PLA","source":"marketplace"}]`)

func geminiQueries(queries string) string {
	return `{"candidates":[{"content":{"role":"model","parts":[{"text":` + jsonString(`{"queries":`+queries+`}`) + `}]},"finishReason":"STOP"}]}`
}

func TestGeminiSearchResearch_QueriesThenSearchThenReport(t *testing.T) {
	searcher := &fakeSearcher{results: foundOnWeb}
	writer := &fakeGemini{replies: []string{geminiQueriesReply, geminiReportReply([]MarketSource{
		{Title: "Shopee", URL: shopeeShop, Price: 49, Kind: sourceRetail},
		{Title: "ลิงก์ที่ AI แต่งเอง", URL: "https://made-up.example.com", Price: 1},
	})}}

	report, err := geminiSearchResearch(context.Background(), searcher, writer, "สินค้า: พวงกุญแจพิมพ์ 3D")
	if err != nil {
		t.Fatal(err)
	}
	want := "พวงกุญแจ 3D@-shopee.co.th-lazada.co.th | พวงกุญแจ 3D@shopee.co.th | พวงกุญแจ 3D@lazada.co.th | เส้น PLA@"
	if searcher.searched() != want {
		t.Errorf("คำแรกค้นทั้งเว็บไทย + Shopee + Lazada / คำถัดไปทั้งเว็บไทย / คำค้นไม่ซ้ำ ไม่ว่าง ได้ %s", searcher.searched())
	}
	if report.RetailTypical != 229 || len(report.Sources) != 1 || report.Sources[0].URL != shopeeShop {
		t.Errorf("ต้องได้ผลจาก Gemini + แหล่งที่มาเฉพาะลิงก์ที่ค้นเจอจริง ได้ %+v", report)
	}
	if report.SearchSuggestions != "" {
		t.Error("ค้นด้วย Tavily ไม่มีแถบคำค้นของ Google")
	}

	if writer.sent[0].GenerationConfig["responseMimeType"] != "application/json" || !strings.Contains(writer.lastRequestJSON(), "พวงกุญแจ") {
		t.Error("ขั้นคิดคำค้นต้องให้ตอบเป็น JSON จากข้อมูลสินค้า")
	}
	input := writer.lastRequestJSON()
	if !strings.Contains(input, "฿49") || !strings.Contains(input, "["+thaiShopsLabel+"] พวงกุญแจ 3D") || strings.Count(input, shopeeShop) != 2 {
		t.Errorf("ขั้นสุดท้ายต้องได้ข้อความจากหน้าเว็บ + ป้ายแหล่ง + ลิงก์ (ผลซ้ำเหลือหน้าละครั้ง: ในผลค้นหา 1 + รายการลิงก์ 1) ได้ %s", input)
	}
}

func TestGeminiSearchResearch_NothingFound(t *testing.T) {
	writer := &fakeGemini{replies: []string{geminiQueriesReply, geminiReportReply(nil)}}
	report, err := geminiSearchResearch(context.Background(), &fakeSearcher{}, writer, "สินค้า")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(writer.lastRequestJSON(), "ไม่พบผลค้นหา") || len(report.Sources) != 0 {
		t.Errorf("ค้นไม่เจอ ต้องบอก Gemini ว่าไม่พบ และไม่มีแหล่งที่มา ได้ %+v", report.Sources)
	}
}

func TestGeminiSearchResearch_Errors(t *testing.T) {
	noQueries := `{"candidates":[{"content":{"role":"model","parts":[{"text":` + jsonString(`{"queries":[]}`) + `}]},"finishReason":"STOP"}]}`
	noCredit := &TavilyError{Status: tavilyPlanLimit, Message: "plan limit"}

	_, err := geminiSearchResearch(context.Background(), &fakeSearcher{}, &fakeGemini{replies: []string{noQueries}}, "สินค้า")
	if !errors.Is(err, ErrNoReport) {
		t.Errorf("คิดคำค้นไม่ได้ ต้องได้ ErrNoReport ได้ %v", err)
	}
	searcher := &fakeSearcher{err: noCredit}
	_, err = geminiSearchResearch(context.Background(), searcher, &fakeGemini{replies: []string{geminiQueriesReply}}, "สินค้า")
	var tavilyErr *TavilyError
	if !errors.As(err, &tavilyErr) || len(searcher.queries) != 1 {
		t.Errorf("เครดิตค้นเว็บหมด ต้องหยุดทันทีแล้วส่ง TavilyError ต่อ ได้ %v (ค้นไป %d คำ)", err, len(searcher.queries))
	}
}

func TestTavilyClient_Search(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &gotBody)
		w.Write([]byte(`{"query":"x","results":[{"title":"Shopee","url":"` + shopeeShop + `","content":"฿49","score":0.9}],"response_time":0.5}`))
	}))
	defer server.Close()

	client := NewTavily("tvly-test").(*tavilyClient)
	client.baseURL = server.URL
	results, err := client.Search(context.Background(), SearchQuery{Text: "พวงกุญแจ 3D ราคา"})
	if err != nil || len(results) != 1 || results[0].URL != shopeeShop || results[0].Content != "฿49" {
		t.Fatalf("ได้ %+v %v", results, err)
	}
	if gotPath != "/search" || gotAuth != "Bearer tvly-test" {
		t.Errorf("path / key ผิด ได้ %q %q", gotPath, gotAuth)
	}
	if gotBody["query"] != "พวงกุญแจ 3D ราคา" || gotBody["search_depth"] != "basic" || gotBody["country"] != "thailand" || gotBody["api_key"] != nil {
		t.Errorf("body ผิด (ต้องค้นแบบ basic = 1 เครดิต, เน้นไทย, ไม่ใส่ key ใน body) ได้ %v", gotBody)
	}
	if gotBody["include_domains"] != nil || gotBody["exclude_domains"] != nil || gotBody["time_range"] != nil {
		t.Errorf("ไม่กำหนดเว็บ = ค้นทุกเว็บ ได้ %v", gotBody)
	}
}

func TestTavilyClient_SearchExcludeSites(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &gotBody)
		w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()

	client := NewTavily("tvly-test").(*tavilyClient)
	client.baseURL = server.URL
	_, err := client.Search(context.Background(), SearchQuery{Text: "x", ExcludeSites: []string{"shopee.co.th", "lazada.co.th"}})
	excluded, _ := gotBody["exclude_domains"].([]any)
	if err != nil || len(excluded) != 2 || excluded[0] != "shopee.co.th" || gotBody["include_domains"] != nil {
		t.Errorf("ต้องส่ง exclude_domains ไม่ส่ง include_domains ได้ %v %v", gotBody, err)
	}
}

func TestTavilyClient_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(tavilyPlanLimit)
		w.Write([]byte(`{"detail":{"error":"This request exceeds your plan's set usage limit."}}`))
	}))
	defer server.Close()

	client := NewTavily("tvly-test").(*tavilyClient)
	client.baseURL = server.URL
	_, err := client.Search(context.Background(), SearchQuery{Text: "x"})
	tavilyErr, ok := err.(*TavilyError)
	if !ok || tavilyErr.Status != tavilyPlanLimit || !strings.Contains(tavilyErr.Message, "usage limit") {
		t.Errorf("ต้องได้ TavilyError 432 พร้อมข้อความ ได้ %v", err)
	}
}

func TestNewTavily_NoKey(t *testing.T) {
	if NewTavily("") != nil {
		t.Error("ไม่มี key ต้องได้ nil")
	}
}

// newTavilyRouter Gemini ฟรี + Tavily (ตัวปลอมทั้งคู่)
func newTavilyRouter(repo Repository, gemini *fakeGemini, searcher *fakeSearcher) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(repo, fakeShop{}, Config{GeminiKey: "free-key", TavilyKey: "tvly-test", Secret: testSecret})
	h.now = func() time.Time { return testToday }
	h.newGemini = func(apiKey, model string) Gemini { return gemini }
	h.newSearcher = func(apiKey string) WebSearcher { return searcher }
	h.RegisterRoutes(r.Group("/api/v1"), middleware.Auth(testSecret))
	return r
}

func TestHandler_CheckMarket_FreeGeminiWithTavily(t *testing.T) {
	repo := &fakeRepository{}
	gemini := &fakeGemini{replies: []string{geminiQueriesReply, geminiReportReply([]MarketSource{{Title: "Shopee", URL: shopeeShop, Price: 49}})}}
	searcher := &fakeSearcher{results: foundOnWeb}
	rec := sendRequest(newTavilyRouter(repo, gemini, searcher), http.MethodPost, "/api/v1/products/"+caseID+"/market", "")

	if rec.Code != http.StatusCreated {
		t.Fatalf("Gemini ฟรี + Tavily เช็คราคาตลาดได้ ต้องได้ 201 ได้ %d %s", rec.Code, rec.Body.String())
	}
	if repo.saved == nil || repo.saved.Sources[0].URL != shopeeShop || len(searcher.queries) != 4 {
		t.Errorf("ต้องค้นด้วย Tavily แล้วบันทึกผล ได้ %+v (ค้น %d คำ)", repo.saved, len(searcher.queries))
	}
	if !strings.Contains(string(gemini.sent[0].SystemInstruction.Parts[0]), "คำค้น") {
		t.Error("ขั้นแรกต้องให้ Gemini คิดคำค้น")
	}
}

func TestHandler_Status_FreeGeminiWithTavily(t *testing.T) {
	rec := sendRequest(newTavilyRouter(&fakeRepository{}, &fakeGemini{}, &fakeSearcher{}), http.MethodGet, "/api/v1/ai/status", "")
	if !strings.Contains(rec.Body.String(), `"provider":"gemini"`) || !strings.Contains(rec.Body.String(), `"web_search":true`) {
		t.Errorf("มี TAVILY_API_KEY → Gemini ฟรีค้นเว็บได้ ได้ %s", rec.Body.String())
	}
}

func TestHandler_CheckMarket_TavilyErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"key ของ Tavily ผิด", &TavilyError{Status: http.StatusUnauthorized}, http.StatusServiceUnavailable, "search_key_invalid"},
		{"เครดิตค้นเว็บเดือนนี้หมด", &TavilyError{Status: tavilyPlanLimit}, http.StatusServiceUnavailable, "search_quota_exceeded"},
		{"ค้นถี่เกิน", &TavilyError{Status: http.StatusTooManyRequests}, http.StatusServiceUnavailable, "search_quota_exceeded"},
		{"Tavily ล่ม", &TavilyError{Status: http.StatusInternalServerError, Message: "boom"}, http.StatusBadGateway, "ai_error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gemini := &fakeGemini{replies: []string{geminiQueriesReply}}
			rec := sendRequest(newTavilyRouter(&fakeRepository{}, gemini, &fakeSearcher{err: tt.err}), http.MethodPost, "/api/v1/products/"+caseID+"/market", "")
			if rec.Code != tt.wantStatus || !strings.Contains(rec.Body.String(), tt.wantCode) {
				t.Errorf("ต้องได้ %d %s ได้ %d %s", tt.wantStatus, tt.wantCode, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestTavilyClient_SitesAndRecent(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &gotBody)
		w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()
	client := NewTavily("tvly-test").(*tavilyClient)
	client.baseURL = server.URL

	client.Search(context.Background(), SearchQuery{Text: "พวงกุญแจ", Sites: []string{"shopee.co.th"}})
	sites, _ := json.Marshal(gotBody["include_domains"])
	if string(sites) != `["shopee.co.th"]` || gotBody["include_domains_mode"] != nil {
		t.Errorf("ค้นเฉพาะเว็บที่กำหนด (ไม่ใส่ mode = เฉพาะเว็บนี้) ได้ %v", gotBody)
	}

	client.Search(context.Background(), SearchQuery{Text: "ราคาหมู", Sites: centralMarketSites, PreferSites: true, Recent: true})
	if gotBody["include_domains_mode"] != "prefer" || gotBody["time_range"] != "month" {
		t.Errorf("ตลาดกลาง: เน้นเว็บที่กำหนด + เฉพาะ 1 เดือนล่าสุด ได้ %v", gotBody)
	}
}
