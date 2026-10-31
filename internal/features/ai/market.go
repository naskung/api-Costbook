package ai

import (
	"context"
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go/v2"
)

// maxResearchRounds ส่งคำขอได้กี่รอบต่อการเช็ค 1 ครั้ง (รอบที่ Claude พักกลางทาง pause_turn + รอบเตือนให้ส่งผล)
const maxResearchRounds = 6

// remindReportText ข้อความเตือนเมื่อ Claude ตอบเป็นข้อความแต่ลืมเรียก save_market_report
const remindReportText = "ส่งผลด้วยเครื่องมือ save_market_report ตอนนี้เลย จากข้อมูลที่ค้นมาแล้ว"

// researchParams คำขอ 1 รอบของงานเช็คราคาตลาด: ค้นเว็บได้ + ส่งผลผ่าน save_market_report
func researchParams(messages []anthropic.BetaMessageParam) anthropic.BetaMessageNewParams {
	return anthropic.BetaMessageNewParams{
		MaxTokens: 16000,
		System: []anthropic.BetaTextBlockParam{{
			Text:         marketSystemPrompt,
			CacheControl: anthropic.NewBetaCacheControlEphemeralParam(),
		}},
		Messages:     messages,
		Tools:        []anthropic.BetaToolUnionParam{webSearchTool, reportTool},
		OutputConfig: anthropic.BetaOutputConfigParam{Effort: anthropic.BetaOutputConfigEffortMedium},
	}
}

// research ให้ Claude ค้นราคาตลาดจากข้อมูลสินค้า (prompt) แล้วคืนผลที่ทำความสะอาดแล้ว
// - pause_turn = Claude ค้นนานจนพักกลางทาง → ส่งประวัติกลับไป Claude ค้นต่อเอง
// - ตอบจบแต่ไม่ส่งผล → เตือน 1 ครั้ง ยังไม่ส่งอีก = ErrNoReport
func research(ctx context.Context, claude Claude, prompt string) (MarketReport, error) {
	messages := []anthropic.BetaMessageParam{anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(prompt))}
	reminded := false

	for round := 0; round < maxResearchRounds; round++ {
		resp, err := claude.Send(ctx, researchParams(messages))
		if err != nil {
			return MarketReport{}, err
		}
		if resp.StopReason == anthropic.BetaStopReasonRefusal {
			return MarketReport{}, ErrRefused
		}
		if report, ok := findReport(resp); ok {
			return cleanReport(report), nil
		}

		messages = append(messages, resp.ToParam())
		if resp.StopReason == anthropic.BetaStopReasonPauseTurn {
			continue
		}
		if reminded {
			return MarketReport{}, ErrNoReport
		}
		reminded = true
		messages = append(messages, anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(remindReportText)))
	}
	return MarketReport{}, ErrNoReport
}

// findReport หา save_market_report ในคำตอบ แล้วอ่านข้อมูลออกมา
func findReport(resp *anthropic.BetaMessage) (MarketReport, bool) {
	for _, block := range resp.Content {
		use, ok := block.AsAny().(anthropic.BetaToolUseBlock)
		if !ok || use.Name != reportToolName {
			continue
		}
		var report MarketReport
		if err := json.Unmarshal([]byte(use.JSON.Input.Raw()), &report); err != nil {
			return MarketReport{}, false
		}
		return report, true
	}
	return MarketReport{}, false
}
