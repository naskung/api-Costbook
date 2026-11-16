package ai

import (
	"context"
	"encoding/json"
	"strings"
)

// geminiResearch เช็คราคาตลาดด้วย Gemini ทำ 2 ขั้น (ค้น Google พร้อมตอบเป็น JSON ในครั้งเดียวไม่ได้)
// ใช้ได้เมื่อเปิด GEMINI_WEB_SEARCH=on (Gemini แบบเสียเงิน — แบบฟรีค้น Google ไม่ได้ ได้ 429)
//  1. search ค้น Google แล้วรายงานราคาที่เจอเป็นข้อความ + ได้ลิงก์ที่ใช้ + แถบคำค้นแนะนำของ Google
//  2. writer ส่งผลค้นหาไปจัดเป็นผลเช็คราคาตลาด (JSON ตาม reportSchema)
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
	report, err := writeReport(ctx, geminiAsker(writer), prompt, findings, groundingSources(grounding))
	if err != nil {
		return MarketReport{}, err
	}
	report.SearchSuggestions = grounding.SearchEntryPoint.RenderedContent
	return report, nil
}

// geminiSearchResearch เช็คราคาตลาดด้วย Gemini ฟรี + ค้นเว็บเอง (Tavily) — Gemini ฟรีค้น Google ไม่ได้
func geminiSearchResearch(ctx context.Context, searcher WebSearcher, writer Gemini, prompt string) (MarketReport, error) {
	return searchResearch(ctx, searcher, geminiAsker(writer), prompt)
}

// geminiAsker ถาม Gemini ครั้งเดียวให้ตอบเป็น JSON ตาม schema (responseMimeType + responseJsonSchema)
func geminiAsker(gemini Gemini) jsonAsker {
	return func(ctx context.Context, system, input string, schema map[string]any) (string, error) {
		resp, err := gemini.Generate(ctx, geminiRequest{
			SystemInstruction: geminiContent{Parts: []json.RawMessage{geminiText(system)}},
			Contents:          []geminiContent{{Role: "user", Parts: []json.RawMessage{geminiText(input)}}},
			GenerationConfig:  map[string]any{"responseMimeType": "application/json", "responseJsonSchema": schema},
		})
		if err != nil {
			return "", err
		}
		if geminiBlocked(resp) {
			return "", ErrRefused
		}
		return geminiAnswerText(resp.Candidates[0].Content.Parts), nil
	}
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
