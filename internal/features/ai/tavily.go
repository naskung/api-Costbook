package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// tavilyBaseURL ที่อยู่ API ของ Tavily (test เปลี่ยนเป็น server ปลอมได้)
const tavilyBaseURL = "https://api.tavily.com"

// status ที่ Tavily ใช้บอกว่าเครดิตหมด: 432 = เครดิตของแผน (ฟรี 1,000 / เดือน) หมด, 433 = เกินวงเงินจ่ายตามใช้
const (
	tavilyPlanLimit = 432
	tavilyPayLimit  = 433
)

// tavilyMaxResults จำนวนผลต่อ 1 คำค้น / ค้นแบบ basic = ใช้ 1 เครดิตเท่าเดิม ไม่ว่าจะขอกี่ผล (ฟรี 1,000 เครดิตต่อเดือน)
const tavilyMaxResults = 8

// WebSearcher ค้นเว็บ 1 ครั้ง (แยกเป็น interface เพื่อให้ test ใช้ตัวปลอมได้)
type WebSearcher interface {
	Search(ctx context.Context, query SearchQuery) ([]SearchResult, error)
}

// SearchQuery การค้น 1 ครั้ง: Sites = ค้นเฉพาะเว็บเหล่านี้ (ว่าง = ทุกเว็บ), PreferSites = เน้นเว็บเหล่านี้แต่เว็บอื่นก็ได้,
// Recent = เฉพาะข้อมูล 1 เดือนล่าสุด (ราคาของสดเปลี่ยนเร็ว ข่าวเก่าใช้ไม่ได้)
type SearchQuery struct {
	Text        string
	Sites       []string
	PreferSites bool
	Recent      bool
}

// SearchResult ผลค้นหา 1 หน้าเว็บ content = ข้อความส่วนที่ตรงกับคำค้น / Source = ป้ายบอกแหล่ง (ใส่ตอนรวมผล ไม่ได้มาจาก Tavily)
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
	Source  string `json:"-"`
}

// TavilyError error ที่ API ของ Tavily ตอบกลับ (status = HTTP status)
// 401 = key ผิด, 429 = ค้นถี่เกิน, 432 / 433 = เครดิตของเดือนนี้หมด
type TavilyError struct {
	Status  int
	Message string
}

func (e *TavilyError) Error() string {
	return fmt.Sprintf("tavily %d: %s", e.Status, e.Message)
}

type tavilyClient struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewTavily สร้างตัวค้นเว็บ — ไม่มี key = คืน nil (Gemini ฟรีเช็คราคาตลาดไม่ได้)
func NewTavily(apiKey string) WebSearcher {
	if apiKey == "" {
		return nil
	}
	return &tavilyClient{apiKey: apiKey, baseURL: tavilyBaseURL, http: &http.Client{}}
}

// Search POST /search แบบ basic เน้นผลจากไทย (ส่ง key ทาง header ไม่ใส่ใน body / URL)
func (t *tavilyClient) Search(ctx context.Context, query SearchQuery) ([]SearchResult, error) {
	request := map[string]any{
		"query":        query.Text,
		"search_depth": "basic",
		"max_results":  tavilyMaxResults,
		"country":      "thailand",
	}
	if len(query.Sites) > 0 {
		request["include_domains"] = query.Sites
	}
	if len(query.Sites) > 0 && query.PreferSites {
		request["include_domains_mode"] = "prefer"
	}
	if query.Recent {
		request["time_range"] = "month"
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL+"/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	res, err := t.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, &TavilyError{Status: res.StatusCode, Message: tavilyErrorMessage(data)}
	}
	var found struct {
		Results []SearchResult `json:"results"`
	}
	err = json.Unmarshal(data, &found)
	return found.Results, err
}

// tavilyErrorMessage อ่านข้อความ error จาก {"detail": {"error": "..."}}
func tavilyErrorMessage(data []byte) string {
	var body struct {
		Detail struct {
			Error string `json:"error"`
		} `json:"detail"`
	}
	if json.Unmarshal(data, &body) != nil || body.Detail.Error == "" {
		return strings.TrimSpace(string(data))
	}
	return body.Detail.Error
}
