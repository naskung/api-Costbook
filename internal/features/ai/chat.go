package ai

import (
	"context"
	"errors"
	"strings"

	"github.com/anthropics/anthropic-sdk-go/v2"
)

// maxChatRounds ส่งคำขอได้กี่รอบต่อคำถาม 1 ข้อ (แต่ละรอบ AI ใช้เครื่องมือได้) / maxActions เสนองานได้กี่งานต่อคำตอบ
const (
	maxChatRounds = 10
	maxActions    = 5
)

// chatParams คำขอ 1 รอบของแชท — คำสั่งหลัก + เครื่องมือไม่เปลี่ยน (cache ได้) วันที่วันนี้แยกไว้ก้อนหลัง
func chatParams(messages []anthropic.BetaMessageParam, today string) anthropic.BetaMessageNewParams {
	return anthropic.BetaMessageNewParams{
		MaxTokens: 16000,
		System: []anthropic.BetaTextBlockParam{
			{Text: chatSystemPrompt, CacheControl: anthropic.NewBetaCacheControlEphemeralParam()},
			{Text: "วันนี้คือวันที่ " + today},
		},
		Messages:     messages,
		Tools:        claudeTools(chatTools),
		OutputConfig: anthropic.BetaOutputConfigParam{Effort: anthropic.BetaOutputConfigEffortMedium},
	}
}

// historyParams แปลงประวัติแชทจาก Flutter (ข้อความล้วน) เป็นรูปแบบที่ Claude รับ
func historyParams(history []ChatMessage) []anthropic.BetaMessageParam {
	messages := []anthropic.BetaMessageParam{}
	for _, m := range history {
		text := anthropic.NewBetaTextBlock(strings.TrimSpace(m.Content))
		if m.Role == "assistant" {
			messages = append(messages, anthropic.BetaMessageParam{
				OfAssistant: &anthropic.BetaAssistantMessageParam{Content: []anthropic.BetaContentBlockParamUnion{text}},
			})
			continue
		}
		messages = append(messages, anthropic.NewBetaUserMessage(text))
	}
	return messages
}

// chat ตอบคำถามล่าสุดในแชท: วนให้ AI ใช้เครื่องมือ (อ่านข้อมูล / เสนองาน) จนได้คำตอบ
func chat(ctx context.Context, claude Claude, tools toolbox, history []ChatMessage) (ChatReply, error) {
	messages := historyParams(history)
	today := tools.today.Format("2006-01-02")
	reply := ChatReply{Actions: []Action{}}

	for round := 0; round < maxChatRounds; round++ {
		resp, err := claude.Send(ctx, chatParams(messages, today))
		if err != nil {
			return ChatReply{}, err
		}
		if resp.StopReason == anthropic.BetaStopReasonRefusal {
			return ChatReply{}, ErrRefused
		}

		messages = append(messages, resp.ToParam())
		if resp.StopReason != anthropic.BetaStopReasonToolUse {
			reply.Reply = replyText(resp)
			if reply.Reply == "" && len(reply.Actions) == 0 {
				return ChatReply{}, ErrNoReport
			}
			return reply, nil
		}

		results, err := runTools(ctx, tools, resp, &reply)
		if err != nil {
			return ChatReply{}, err
		}
		messages = append(messages, anthropic.NewBetaUserMessage(results...))
	}
	return ChatReply{}, ErrTooManySteps
}

// runTools ทำทุกเครื่องมือที่ AI เรียกในรอบนี้ แล้วส่งผลกลับในข้อความเดียว (AI เรียกหลายตัวพร้อมกันได้)
// งานที่เสนอเก็บไว้ใน reply.Actions (ไม่เกิน maxActions)
func runTools(ctx context.Context, tools toolbox, resp *anthropic.BetaMessage, reply *ChatReply) ([]anthropic.BetaContentBlockParamUnion, error) {
	results := []anthropic.BetaContentBlockParamUnion{}
	for _, block := range resp.Content {
		use, ok := block.AsAny().(anthropic.BetaToolUseBlock)
		if !ok {
			continue
		}
		if strings.HasPrefix(use.Name, "propose_") && len(reply.Actions) >= maxActions {
			results = append(results, anthropic.NewBetaToolResultBlock(use.ID, "เสนอได้ไม่เกิน 5 งานต่อครั้ง", true))
			continue
		}

		text, action, err := tools.run(ctx, use.Name, []byte(use.JSON.Input.Raw()))
		var toolErr toolError
		if errors.As(err, &toolErr) {
			results = append(results, anthropic.NewBetaToolResultBlock(use.ID, toolErr.Error(), true))
			continue
		}
		if err != nil {
			return nil, err
		}
		if action != nil {
			action.ID = use.ID
			reply.Actions = append(reply.Actions, *action)
		}
		results = append(results, anthropic.NewBetaToolResultBlock(use.ID, text, false))
	}
	return results, nil
}

// replyText รวมข้อความทุกก้อนในคำตอบของ AI
func replyText(resp *anthropic.BetaMessage) string {
	parts := []string{}
	for _, block := range resp.Content {
		if text, ok := block.AsAny().(anthropic.BetaTextBlock); ok && strings.TrimSpace(text.Text) != "" {
			parts = append(parts, strings.TrimSpace(text.Text))
		}
	}
	return strings.Join(parts, "\n\n")
}
