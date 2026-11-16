package ai

import (
	"context"
	"encoding/json"
	"strings"
)

// maxSearchSnippet ตัดข้อความแต่ละหน้าไม่เกินกี่ตัวอักษร
const maxSearchSnippet = 600

// jsonAsker ถาม AI 1 ครั้งให้ตอบเป็น JSON ตาม schema ได้ข้อความ JSON กลับ
// Gemini (geminiAsker) และ AI แบบ OpenAI (openAIAsker) ใช้ขั้นตอนเช็คราคาตลาดแบบค้นเว็บเองชุดเดียวกัน
type jsonAsker func(ctx context.Context, system, input string, schema map[string]any) (string, error)

// querySchema รูปแบบ JSON ของคำค้นที่ให้ AI คิด: คำค้น + แหล่งที่ควรค้น
var querySchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"queries": map[string]any{
			"type":        "array",
			"description": "คำค้น 2–3 คำ",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"text":   map[string]any{"type": "string", "description": "คำค้น"},
					"source": map[string]any{"type": "string", "enum": []string{searchMarketplace, searchCentralMarket}},
				},
				"required": []string{"text", "source"},
			},
		},
	},
	"required": []string{"queries"},
}

// searchResearch เช็คราคาตลาดด้วย AI + ค้นเว็บเอง (Tavily ฟรี 1,000 ครั้ง/เดือน) — ใช้กับ AI ที่ค้นเว็บเองไม่ได้
//  1. AI คิดคำค้น 2–3 คำ + เลือกแหล่ง: ร้านค้าในไทย (ทั้งเว็บไทย + Shopee / Lazada) หรือตลาดกลาง (ของสด)
//  2. searcher ค้นตาม searchPlan (ครั้งละ 1 เครดิต ไม่เกิน maxSearches) รวมผลที่ไม่ซ้ำ พร้อมป้ายบอกแหล่ง
//  3. AI จัดผลค้นหาเป็นผลเช็คราคาตลาด — แหล่งที่มาเฉพาะลิงก์ที่ค้นเจอจริง
func searchResearch(ctx context.Context, searcher WebSearcher, ask jsonAsker, prompt string) (MarketReport, error) {
	queries, err := searchQueries(ctx, ask, prompt)
	if err != nil {
		return MarketReport{}, err
	}
	results := []SearchResult{}
	seen := map[string]bool{}
	for _, search := range searchPlan(queries) {
		found, err := searcher.Search(ctx, search.query)
		if err != nil {
			return MarketReport{}, err
		}
		for _, result := range found {
			if result.URL == "" || seen[result.URL] {
				continue
			}
			seen[result.URL] = true
			result.Source = search.label
			results = append(results, result)
		}
	}
	return writeReport(ctx, ask, prompt, searchFindings(results), searchSources(results))
}

// searchQueries ให้ AI คิดคำค้น + แหล่งจากข้อมูลสินค้า (ตอบเป็น JSON) — คำค้นที่ใช้ได้ไม่ซ้ำ
func searchQueries(ctx context.Context, ask jsonAsker, prompt string) ([]plannedQuery, error) {
	text, err := ask(ctx, queryPrompt, prompt, querySchema)
	if err != nil {
		return nil, err
	}
	var answer struct {
		Queries []plannedQuery `json:"queries"`
	}
	if json.Unmarshal([]byte(text), &answer) != nil {
		return nil, ErrNoReport
	}
	queries := cleanQueries(answer.Queries)
	if len(queries) == 0 {
		return nil, ErrNoReport
	}
	return queries, nil
}

// writeReport ขั้นสุดท้าย: ส่งข้อมูลสินค้า + ผลค้นหา + ลิงก์ ให้ AI จัดเป็นผลเช็คราคาตลาด (JSON ตาม reportSchema)
// แหล่งที่มาเก็บเฉพาะลิงก์ที่ค้นเจอจริง (AI แต่งลิงก์เองไม่ได้)
func writeReport(ctx context.Context, ask jsonAsker, prompt, findings string, links []MarketSource) (MarketReport, error) {
	text, err := ask(ctx, reportPrompt, reportInput(prompt, findings, links), reportSchema)
	if err != nil {
		return MarketReport{}, err
	}
	var report MarketReport
	if json.Unmarshal([]byte(text), &report) != nil {
		return MarketReport{}, ErrNoReport
	}
	report.Sources = foundSourcesOnly(report.Sources, links)
	return cleanReport(report), nil
}

// searchFindings ผลค้นหาเป็นข้อความให้ AI อ่าน: [แหล่ง] ชื่อหน้า (ลิงก์): ข้อความ (ตัดแต่ละหน้าไม่ให้ยาวเกิน)
func searchFindings(results []SearchResult) string {
	if len(results) == 0 {
		return "ไม่พบผลค้นหา"
	}
	lines := []string{}
	for _, r := range results {
		lines = append(lines, "- ["+r.Source+"] "+r.Title+" ("+r.URL+"): "+shortText(r.Content, maxSearchSnippet))
	}
	return strings.Join(lines, "\n")
}

// searchSources ลิงก์ที่ค้นเจอ ไว้ให้ AI เลือกเป็นแหล่งที่มา
func searchSources(results []SearchResult) []MarketSource {
	sources := []MarketSource{}
	for _, r := range results {
		sources = append(sources, MarketSource{Title: r.Title, URL: r.URL, Kind: sourceRetail})
	}
	return sources
}

// reportInput ข้อความที่ส่งให้ขั้นสุดท้าย: ข้อมูลสินค้า + ผลค้นหา + รายการแหล่งที่มาที่เลือกได้
func reportInput(prompt, findings string, links []MarketSource) string {
	lines := []string{prompt, "", "ผลค้นหา:", findings, "", "แหล่งที่มา:"}
	for _, link := range links {
		lines = append(lines, "- "+link.Title+": "+link.URL)
	}
	if len(links) == 0 {
		lines = append(lines, "- (ไม่มี)")
	}
	return strings.Join(lines, "\n")
}

// foundSourcesOnly เก็บเฉพาะแหล่งที่มาที่เป็นลิงก์ที่ค้นเจอจริง — ไม่เหลือเลย = ใช้ลิงก์ที่ค้นเจอทั้งหมด (ไม่มีราคา)
func foundSourcesOnly(reported, links []MarketSource) []MarketSource {
	found := map[string]bool{}
	for _, link := range links {
		found[link.URL] = true
	}
	sources := []MarketSource{}
	for _, s := range reported {
		if found[s.URL] {
			sources = append(sources, s)
		}
	}
	if len(sources) == 0 {
		return links
	}
	return sources
}
