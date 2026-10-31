package ai

import (
	"strings"
	"testing"
)

func TestChatRequest_Validate(t *testing.T) {
	user := ChatMessage{Role: "user", Content: "ยอดขายเดือนนี้เท่าไหร่"}
	assistant := ChatMessage{Role: "assistant", Content: "1,450 บาทครับ"}

	tests := []struct {
		name     string
		messages []ChatMessage
		want     string
	}{
		{"ถามข้อแรก ผ่าน", []ChatMessage{user}, ""},
		{"ถาม ตอบ ถามต่อ ผ่าน", []ChatMessage{user, assistant, user}, ""},
		{"ไม่มีข้อความ", []ChatMessage{}, "ai_messages_invalid"},
		{"จบด้วยคำตอบของ AI (ไม่มีคำถามใหม่)", []ChatMessage{user, assistant}, "ai_messages_invalid"},
		{"เริ่มด้วย assistant", []ChatMessage{assistant, user, user}, "ai_messages_invalid"},
		{"user ซ้อนกัน 2 อัน", []ChatMessage{user, user, user}, "ai_messages_invalid"},
		{"ข้อความว่าง", []ChatMessage{{Role: "user", Content: "  "}}, "ai_messages_invalid"},
		{"role แปลก ๆ", []ChatMessage{{Role: "system", Content: "x"}}, "ai_messages_invalid"},
		{"ข้อความยาวเกิน 4000 ตัวอักษร", []ChatMessage{{Role: "user", Content: strings.Repeat("ก", 4001)}}, "ai_message_too_long"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := (ChatRequest{Messages: tt.messages}).Validate(); got != tt.want {
				t.Errorf("ได้ %q ต้องได้ %q", got, tt.want)
			}
		})
	}
}

func TestChatRequest_Validate_TooMany(t *testing.T) {
	messages := []ChatMessage{}
	for i := 0; i < maxChatMessages+1; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		messages = append(messages, ChatMessage{Role: role, Content: "x"})
	}
	if got := (ChatRequest{Messages: messages}).Validate(); got != "ai_messages_invalid" {
		t.Errorf("41 ข้อความ ได้ %q ต้องได้ ai_messages_invalid", got)
	}
}
