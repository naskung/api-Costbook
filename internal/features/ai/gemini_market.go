package ai

import (
	"context"
	"encoding/json"
	"strings"
)

// maxSearchSnippet ตัดข้อความแต่ละหน้าไม่เกินกี่ตัวอักษร
const maxSearchSnippet = 600

// geminiQuerySchema รูปแบบ JSON ของคำค้นที่ให้ Gemini คิด: คำค้น + แหล่งที่ควรค้น
var geminiQuerySchema = map[string]any{
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

// geminiResearch เช็คราคาตลาดด้วย Gemini ทำ 2 ขั้น (ค้น Google พร้อมตอบเป็น JSON ในครั้งเดียวไม่ได้)
// ใช้ได้เมื่อเปิด GEMINI_WEB_SEARCH=on (Gemini แบบเสียเงิน — แบบฟรีค้น Google ไม่ได้ ได้ 429)
//  1. search ค้น Google แล้วรายงานราคาที่เจอเป็นข้อความ + ได้ลิงก์ที่ใช้ + แถบคำค้นแนะนำของ Google
//  2. writer ส่งผลค้นหาไปจัดเป็นผลเช็คราคาตลาด (JSON ตาม geminiReportSchema)
//
// แหล่งที่มาใช้เฉพาะลิงก์ที่ Google ค้นเจอจริง (AI แต่งลิงก์เองไม่ได้) / แถบคำค้นแนะนำเก็บไว้แสดงคู่กับผล (กติกาของ Google)
func geminiResearch(ctx context.Context, search, writer Gemini, prompt string) (MarketReport, error) {
	found, err := search.Generate(ctx, geminiRequest{
		SystemInstruction: geminiContent{Parts: []json.RawMessage{geminiText(geminiSearchPrompt)}},
		Contents:          []geminiContent{{Role: "user", Parts: []json.RawMessage{geminiText(prompt)}}},
		Tools:             []map[string]any{{"google_search": map[string]any{}}},
	})
	if err != nil {
		return MarketReport{}, err
	}
	if geminiBlocked(found) {
		return MarketReport{}, ErrRefused
	}
	findings := geminiAnswerText(found.Candidates[0].Content.Parts)
	if findings == "" {
		return MarketReport{}, ErrNoReport
	}
	grounding := found.Candidates[0].GroundingMetadata
	report, err := geminiWriteReport(ctx, writer, prompt, findings, groundingSources(grounding))
	if err != nil {
		return MarketReport{}, err
	}
	report.SearchSuggestions = grounding.SearchEntryPoint.RenderedContent
	return report, nil
}

// geminiSearchResearch เช็คราคาตลาดด้วย Gemini ฟรี + ค้นเว็บเอง (Tavily ฟรี 1,000 ครั้ง/เดือน) — Gemini ฟรีค้น Google ไม่ได้
//  1. writer คิดคำค้น 2–3 คำ + เลือกแหล่ง: ร้านออนไลน์ (Shopee + Lazada) หรือตลาดกลาง (ของสด)
//  2. searcher ค้นตาม searchPlan (ครั้งละ 1 เครดิต ไม่เกิน maxSearches) รวมผลที่ไม่ซ้ำ พร้อมป้ายบอกแหล่ง
//  3. writer จัดผลค้นหาเป็นผลเช็คราคาตลาด — แหล่งที่มาเฉพาะลิงก์ที่ค้นเจอจริง
func geminiSearchResearch(ctx context.Context, searcher WebSearcher, writer Gemini, prompt string) (MarketReport, error) {
	queries, err := geminiSearchQueries(ctx, writer, prompt)
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
	return geminiWriteReport(ctx, writer, prompt, searchFindings(results), searchSources(results))
}

// geminiSearchQueries ให้ Gemini คิดคำค้น + แหล่งจากข้อมูลสินค้า (ตอบเป็น JSON) — คำค้นที่ใช้ได้ไม่ซ้ำ
func geminiSearchQueries(ctx context.Context, writer Gemini, prompt string) ([]plannedQuery, error) {
	resp, err := writer.Generate(ctx, geminiRequest{
		SystemInstruction: geminiContent{Parts: []json.RawMessage{geminiText(geminiQueryPrompt)}},
		Contents:          []geminiContent{{Role: "user", Parts: []json.RawMessage{geminiText(prompt)}}},
		GenerationConfig:  map[string]any{"responseMimeType": "application/json", "responseJsonSchema": geminiQuerySchema},
	})
	if err != nil {
		return nil, err
	}
	if geminiBlocked(resp) {
		return nil, ErrRefused
	}
	var answer struct {
		Queries []plannedQuery `json:"queries"`
	}
	if json.Unmarshal([]byte(geminiAnswerText(resp.Candidates[0].Content.Parts)), &answer) != nil {
		return nil, ErrNoReport
	}
	queries := cleanQueries(answer.Queries)
	if len(queries) == 0 {
		return nil, ErrNoReport
	}
	return queries, nil
}

// searchFindings ผลค้นหาเป็นข้อความให้ Gemini อ่าน: [แหล่ง] ชื่อหน้า (ลิงก์): ข้อความ (ตัดแต่ละหน้าไม่ให้ยาวเกิน)
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

// searchSources ลิงก์ที่ค้นเจอ ไว้ให้ Gemini เลือกเป็นแหล่งที่มา
func searchSources(results []SearchResult) []MarketSource {
	sources := []MarketSource{}
	for _, r := range results {
		sources = append(sources, MarketSource{Title: r.Title, URL: r.URL, Kind: sourceRetail})
	}
	return sources
}

// geminiWriteReport ขั้นสุดท้าย: ส่งข้อมูลสินค้า + ผลค้นหา + ลิงก์ ให้ Gemini จัดเป็นผลเช็คราคาตลาด (JSON ตาม geminiReportSchema)
// แหล่งที่มาเก็บเฉพาะลิงก์ที่ค้นเจอจริง (AI แต่งลิงก์เองไม่ได้)
func geminiWriteReport(ctx context.Context, writer Gemini, prompt, findings string, links []MarketSource) (MarketReport, error) {
	written, err := writer.Generate(ctx, geminiRequest{
		SystemInstruction: geminiContent{Parts: []json.RawMessage{geminiText(geminiReportPrompt)}},
		Contents:          []geminiContent{{Role: "user", Parts: []json.RawMessage{geminiText(reportInput(prompt, findings, links))}}},
		GenerationConfig:  map[string]any{"responseMimeType": "application/json", "responseJsonSchema": geminiReportSchema},
	})
	if err != nil {
		return MarketReport{}, err
	}
	if geminiBlocked(written) {
		return MarketReport{}, ErrRefused
	}
	var report MarketReport
	if json.Unmarshal([]byte(geminiAnswerText(written.Candidates[0].Content.Parts)), &report) != nil {
		return MarketReport{}, ErrNoReport
	}
	report.Sources = foundSourcesOnly(report.Sources, links)
	return cleanReport(report), nil
}

// geminiAnswerText ข้อความคำตอบ (ไม่รวมส่วนที่ Gemini คิดในใจ)
func geminiAnswerText(parts []json.RawMessage) string {
	texts := []string{}
	for _, raw := range parts {
		var part geminiPart
		if json.Unmarshal(raw, &part) != nil || part.Thought || strings.TrimSpace(part.Text) == "" {
			continue
		}
		texts = append(texts, strings.TrimSpace(part.Text))
	}
	return strings.Join(texts, "\n")
}

// groundingSources หน้าเว็บที่ Google ค้นเจอ (ไม่เอาลิงก์ซ้ำ) — title ส่วนใหญ่เป็นชื่อเว็บ เช่น shopee.co.th
func groundingSources(grounding geminiGrounding) []MarketSource {
	sources := []MarketSource{}
	seen := map[string]bool{}
	for _, chunk := range grounding.GroundingChunks {
		url := chunk.Web.URI
		if url == "" || seen[url] {
			continue
		}
		seen[url] = true
		sources = append(sources, MarketSource{Title: chunk.Web.Title, URL: url, Kind: sourceRetail})
	}
	return sources
}

// reportInput ข้อความที่ส่งให้ขั้นที่ 2: ข้อมูลสินค้า + ผลค้นหา + รายการแหล่งที่มาที่เลือกได้
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

// foundSourcesOnly เก็บเฉพาะแหล่งที่มาที่เป็นลิงก์ที่ Google ค้นเจอจริง — ไม่เหลือเลย = ใช้ลิงก์ที่ค้นเจอทั้งหมด (ไม่มีราคา)
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
