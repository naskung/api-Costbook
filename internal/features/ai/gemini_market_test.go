package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/middleware"
)

// ลิงก์ที่ Google ค้นเจอ (รูปแบบเดียวกับของจริง: ลิงก์ผ่านหน้า redirect ของ Google)
const (
	shopeeLink      = "https://vertexaisearch.cloud.google.com/grounding-api-redirect/AAA"
	lazadaLink      = "https://vertexaisearch.cloud.google.com/grounding-api-redirect/BBB"
	testSuggestions = `<div class="container"><a class="chip" href="https://www.google.com/search?q=x">เคสโทรศัพท์ ราคา</a></div>`
)

// geminiSearchReply ขั้นที่ 1: ค้น Google แล้วรายงานเป็นข้อความ + ลิงก์ที่ใช้ (Shopee ซ้ำ 2 ครั้ง) + แถบคำค้นแนะนำ
var geminiSearchReply = `{"candidates":[{"content":{"role":"model","parts":[{"text":"Shopee: เคสโทรศัพท์ 199 บาท/ชิ้น\nLazada: 259 บาท/ชิ้น"}]},"finishReason":"STOP",` +
	`"groundingMetadata":{"groundingChunks":[{"web":{"uri":"` + shopeeLink + `","title":"shopee.co.th"}},{"web":{"uri":"` + lazadaLink + `","title":"lazada.co.th"}},{"web":{"uri":"` + shopeeLink + `","title":"shopee.co.th"}}],` +
	`"searchEntryPoint":{"renderedContent":` + jsonString(testSuggestions) + `}}}]}`

// geminiReportReply ขั้นที่ 2: ตอบผลเป็น JSON (เป็นข้อความใน part เหมือนของจริง)
func geminiReportReply(sources []MarketSource) string {
	report := MarketReport{
		Summary: "ราคาในตลาด 199–259 บาท", RetailLow: 199, RetailHigh: 259, RetailTypical: 229,
		SuggestedPrice: 239, Advice: "ตั้งราคาใกล้ค่ากลาง", Confidence: confidenceMid, Sources: sources,
	}
	data, _ := json.Marshal(report)
	return `{"candidates":[{"content":{"role":"model","parts":[{"text":` + jsonString(string(data)) + `}]},"finishReason":"STOP"}]}`
}

func jsonString(text string) string {
	data, _ := json.Marshal(text)
	return string(data)
}

func TestGeminiResearch_SearchThenReport(t *testing.T) {
	search := &fakeGemini{replies: []string{geminiSearchReply}}
	writer := &fakeGemini{replies: []string{geminiReportReply([]MarketSource{
		{Title: "Shopee", URL: shopeeLink, Price: 199, Kind: sourceRetail},
		{Title: "เว็บที่ AI แต่งเอง", URL: "https://made-up.example.com", Price: 1, Kind: sourceRetail},
	})}}

	report, err := geminiResearch(context.Background(), search, writer, "สินค้า: เคสโทรศัพท์")
	if err != nil {
		t.Fatal(err)
	}
	if report.RetailTypical != 229 || report.SuggestedPrice != 239 || report.Confidence != confidenceMid {
		t.Errorf("ต้องได้ตัวเลขจากขั้นที่ 2 ได้ %+v", report)
	}
	if len(report.Sources) != 1 || report.Sources[0].URL != shopeeLink || report.Sources[0].Price != 199 {
		t.Errorf("แหล่งที่มาต้องเหลือเฉพาะลิงก์ที่ Google ค้นเจอจริง ได้ %+v", report.Sources)
	}
	if report.SearchSuggestions != testSuggestions {
		t.Errorf("ต้องเก็บแถบคำค้นแนะนำของ Google ไว้แสดงคู่กับผล ได้ %q", report.SearchSuggestions)
	}

	searchTools, _ := json.Marshal(search.sent[0].Tools)
	if !strings.Contains(string(searchTools), `"google_search"`) {
		t.Errorf("ขั้นที่ 1 ต้องเปิดค้น Google ได้ %s", searchTools)
	}
	if writer.sent[0].GenerationConfig["responseMimeType"] != "application/json" || len(writer.sent[0].Tools) != 0 {
		t.Errorf("ขั้นที่ 2 ต้องให้ตอบเป็น JSON ไม่ค้นซ้ำ ได้ %+v", writer.sent[0])
	}
	input := writer.lastRequestJSON()
	if !strings.Contains(input, "Lazada: 259") || !strings.Contains(input, lazadaLink) || !strings.Contains(input, "เคสโทรศัพท์") {
		t.Errorf("ขั้นที่ 2 ต้องได้ข้อมูลสินค้า + ผลค้นหา + ลิงก์ ได้ %s", input)
	}
}

func TestGeminiResearch_NoGoodSources_UsesFoundLinks(t *testing.T) {
	search := &fakeGemini{replies: []string{geminiSearchReply}}
	writer := &fakeGemini{replies: []string{geminiReportReply([]MarketSource{{Title: "แต่งเอง", URL: "https://made-up.example.com"}})}}

	report, err := geminiResearch(context.Background(), search, writer, "สินค้า")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Sources) != 2 || report.Sources[0].URL != shopeeLink || report.Sources[1].URL != lazadaLink {
		t.Errorf("AI ไม่ได้เลือกลิงก์ที่ค้นเจอ → ใช้ลิงก์ที่ Google ค้นเจอทั้งหมด (ไม่ซ้ำ) ได้ %+v", report.Sources)
	}
}

func TestGeminiResearch_Errors(t *testing.T) {
	emptySearch := `{"candidates":[{"content":{"role":"model","parts":[{"text":"คิดอยู่","thought":true}]},"finishReason":"STOP"}]}`
	notJSON := `{"candidates":[{"content":{"role":"model","parts":[{"text":"ไม่ใช่ JSON"}]},"finishReason":"STOP"}]}`
	quota := &GeminiError{Status: http.StatusTooManyRequests, Message: "quota"}
	tests := []struct {
		name   string
		search *fakeGemini
		writer *fakeGemini
		want   error
	}{
		{"ค้นแล้วไม่ได้ข้อความ", &fakeGemini{replies: []string{emptySearch}}, &fakeGemini{}, ErrNoReport},
		{"Google ไม่ยอมตอบ", &fakeGemini{replies: []string{geminiBlockedReply}}, &fakeGemini{}, ErrRefused},
		{"ขั้นที่ 2 ตอบไม่เป็น JSON", &fakeGemini{replies: []string{geminiSearchReply}}, &fakeGemini{replies: []string{notJSON}}, ErrNoReport},
		{"ขั้นที่ 2 ไม่ยอมตอบ", &fakeGemini{replies: []string{geminiSearchReply}}, &fakeGemini{replies: []string{geminiBlockedReply}}, ErrRefused},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := geminiResearch(context.Background(), tt.search, tt.writer, "สินค้า")
			if !errors.Is(err, tt.want) {
				t.Errorf("ต้องได้ %v ได้ %v", tt.want, err)
			}
		})
	}

	t.Run("โควตาค้น Google ฟรีหมด ส่ง error ของ Gemini ต่อ", func(t *testing.T) {
		_, err := geminiResearch(context.Background(), &fakeGemini{err: quota}, &fakeGemini{}, "สินค้า")
		var geminiErr *GeminiError
		if !errors.As(err, &geminiErr) || geminiErr.Status != http.StatusTooManyRequests {
			t.Errorf("ต้องได้ GeminiError 429 ได้ %v", err)
		}
	})
}

// newGeminiModelRouter เหมือน newGeminiRouter แต่จำว่าขอ Gemini รุ่นไหน + แยกตัวค้น / ตัวจัดผลตามรุ่น
func newGeminiModelRouter(repo Repository, cfg Config, byModel map[string]*fakeGemini, models *[]string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(repo, fakeShop{}, cfg)
	h.now = func() time.Time { return testToday }
	h.newGemini = func(apiKey, model string) Gemini {
		*models = append(*models, model)
		return byModel[model]
	}
	h.RegisterRoutes(r.Group("/api/v1"), middleware.Auth(testSecret))
	return r
}

func TestHandler_CheckMarket_GeminiWebSearch(t *testing.T) {
	repo := &fakeRepository{}
	var models []string
	byModel := map[string]*fakeGemini{
		DefaultGeminiModel: {replies: []string{geminiSearchReply, geminiReportReply([]MarketSource{{Title: "Lazada", URL: lazadaLink, Price: 259}})}},
	}
	cfg := Config{GeminiKey: "paid-key", GeminiWebSearch: true, Secret: testSecret}
	rec := sendRequest(newGeminiModelRouter(repo, cfg, byModel, &models), http.MethodPost, "/api/v1/products/"+caseID+"/market", "")

	if rec.Code != http.StatusCreated {
		t.Fatalf("เปิด GEMINI_WEB_SEARCH แล้วเช็คราคาตลาดได้ ต้องได้ 201 ได้ %d %s", rec.Code, rec.Body.String())
	}
	if repo.saved == nil || repo.saved.SearchSuggestions != testSuggestions || repo.saved.RetailTypical != 229 {
		t.Errorf("ต้องบันทึกผล + แถบคำค้นแนะนำ ได้ %+v", repo.saved)
	}
	if !strings.Contains(rec.Body.String(), `"search_suggestions"`) {
		t.Errorf("ต้องส่งแถบคำค้นแนะนำให้ Flutter แสดง ได้ %s", rec.Body.String())
	}
}

func TestHandler_CheckMarket_GeminiSearchModelFromEnv(t *testing.T) {
	var models []string
	byModel := map[string]*fakeGemini{
		"gemini-9-search":  {replies: []string{geminiSearchReply}},
		DefaultGeminiModel: {replies: []string{geminiReportReply(nil)}},
	}
	cfg := Config{GeminiKey: "paid-key", GeminiWebSearch: true, GeminiSearchModel: "gemini-9-search", Secret: testSecret}
	rec := sendRequest(newGeminiModelRouter(&fakeRepository{}, cfg, byModel, &models), http.MethodPost, "/api/v1/products/"+caseID+"/market", "")

	if rec.Code != http.StatusCreated || !strings.Contains(strings.Join(models, ","), "gemini-9-search") {
		t.Errorf("GEMINI_SEARCH_MODEL ต้องเปลี่ยนรุ่นที่ค้นได้ ได้ %d %v", rec.Code, models)
	}
}
