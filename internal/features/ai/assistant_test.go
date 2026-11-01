package ai

import (
	"net/http"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/naskung/api-Costbook/internal/utils"
)

const ownKey = "sk-ant-api03-shop-key-a1B2"

func TestPickProvider(t *testing.T) {
	both := Config{ClaudeKey: envClaudeKey, GeminiKey: "free"}
	tests := []struct {
		name         string
		savedKey     string
		cfg          Config
		wantProvider string
		wantSource   string
		wantSearch   bool
	}{
		{"ร้านใส่ key เอง ใช้ key ของร้านก่อน", ownKey, both, providerClaude, sourceSettings, true},
		{"ไม่มี key ของร้าน ใช้ key ใน .env", "", both, providerClaude, sourceEnv, true},
		{"ไม่มี key Claude เลย ใช้ Gemini ฟรี (ค้น Google ไม่ได้)", "", Config{GeminiKey: "free"}, providerGemini, sourceEnv, false},
		{"Gemini เปิด GEMINI_WEB_SEARCH (แบบเสียเงิน) ค้น Google ได้", "", Config{GeminiKey: "paid", GeminiWebSearch: true}, providerGemini, sourceEnv, true},
		{"ไม่มีอะไรเลย", "", Config{}, providerNone, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pickProvider(tt.savedKey, tt.cfg)
			if got.Provider != tt.wantProvider || got.Source != tt.wantSource || got.WebSearch != tt.wantSearch {
				t.Errorf("ได้ %+v", got)
			}
			if got.Enabled != (tt.wantProvider != providerNone) {
				t.Errorf("enabled ผิด ได้ %+v", got)
			}
		})
	}
}

func TestPickProvider_ModelAndHint(t *testing.T) {
	free := pickProvider("", Config{GeminiKey: "free"})
	if free.Model != DefaultGeminiModel || !free.FreeAvailable || free.HasOwnKey {
		t.Errorf("Gemini ฟรีไม่ตั้งรุ่น = รุ่นเริ่มต้น ได้ %+v", free)
	}
	custom := pickProvider("", Config{GeminiKey: "free", GeminiModel: "gemini-3.5-flash-lite"})
	if custom.Model != "gemini-3.5-flash-lite" {
		t.Errorf("ตั้ง GEMINI_MODEL แล้วต้องใช้รุ่นนั้น ได้ %q", custom.Model)
	}
	own := pickProvider(ownKey, Config{})
	if own.Model != model || own.KeyHint != "…a1B2" || !own.HasOwnKey {
		t.Errorf("key ของร้าน ต้องบอก 4 ตัวท้าย ได้ %+v", own)
	}
}

func TestValidClaudeKey(t *testing.T) {
	tests := map[string]bool{
		ownKey:                   true,
		"sk-ant-short":           false,
		"sk-proj-abcdefghijklm":  false,
		"sk-ant-api03 has space": false,
		"":                       false,
	}
	for key, want := range tests {
		if got := validClaudeKey(key); got != want {
			t.Errorf("validClaudeKey(%q) = %v ต้องได้ %v", key, got, want)
		}
	}
}

func TestHandler_SaveKey(t *testing.T) {
	repo := &fakeRepository{}
	router := newRouterWith(repo, Config{GeminiKey: "free", Secret: testSecret}, &fakeClaude{}, &fakeGemini{})

	rec := sendRequest(router, http.MethodPut, "/api/v1/ai/key", `{"api_key":"  `+ownKey+`  "}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"provider":"claude"`) || !strings.Contains(body, `"source":"settings"`) || !strings.Contains(body, `"key_hint":"…a1B2"`) {
		t.Errorf("บันทึกแล้วต้องเปลี่ยนเป็น Claude ของร้าน ได้ %s", body)
	}
	if strings.Contains(body, ownKey) || strings.Contains(repo.encryptedKey, "sk-ant") {
		t.Error("ห้ามส่ง key กลับ และห้ามเก็บ key เป็นข้อความตรง ๆ")
	}
	plain, _ := utils.Decrypt(repo.encryptedKey, testSecret)
	if plain != ownKey {
		t.Errorf("ถอดรหัสแล้วต้องได้ key เดิม (ตัดช่องว่างแล้ว) ได้ %q", plain)
	}

	rec = sendRequest(router, http.MethodDelete, "/api/v1/ai/key", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"provider":"gemini"`) || repo.encryptedKey != "" {
		t.Errorf("ลบ key แล้วต้องกลับไปใช้ Gemini ฟรี ได้ %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_SaveKey_Errors(t *testing.T) {
	tests := []struct {
		name       string
		claude     *fakeClaude
		body       string
		wantStatus int
		wantBody   string
	}{
		{"JSON พัง", &fakeClaude{}, `{`, http.StatusBadRequest, `{"error":"invalid_json"}`},
		{"รูปแบบ key ผิด (ไม่ใช่ของ Claude)", &fakeClaude{}, `{"api_key":"sk-proj-123456789012345"}`, http.StatusBadRequest, `{"error":"ai_key_format_invalid"}`},
		{"Claude บอกว่า key ใช้ไม่ได้", &fakeClaude{checkErr: &anthropic.Error{StatusCode: http.StatusUnauthorized}}, `{"api_key":"` + ownKey + `"}`, http.StatusBadRequest, `{"error":"ai_key_invalid"}`},
		{"Claude ไม่ว่างตอนลอง key", &fakeClaude{checkErr: &anthropic.Error{StatusCode: http.StatusServiceUnavailable}}, `{"api_key":"` + ownKey + `"}`, http.StatusServiceUnavailable, `{"error":"ai_busy"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepository{}
			router := newRouterWith(repo, Config{Secret: testSecret}, tt.claude, nil)
			rec := sendRequest(router, http.MethodPut, "/api/v1/ai/key", tt.body)
			if rec.Code != tt.wantStatus || strings.TrimSpace(rec.Body.String()) != tt.wantBody {
				t.Errorf("ได้ %d %s ต้องได้ %d %s", rec.Code, rec.Body.String(), tt.wantStatus, tt.wantBody)
			}
			if repo.encryptedKey != "" {
				t.Error("key ใช้ไม่ได้ ต้องไม่บันทึก")
			}
		})
	}
}

func TestHandler_SavedKeyBrokenSecret(t *testing.T) {
	encrypted, _ := utils.Encrypt(ownKey, "old-secret")
	repo := &fakeRepository{encryptedKey: encrypted}
	router := newRouterWith(repo, Config{GeminiKey: "free", Secret: testSecret}, &fakeClaude{}, &fakeGemini{})
	rec := sendRequest(router, http.MethodGet, "/api/v1/ai/status", "")
	if !strings.Contains(rec.Body.String(), `"provider":"gemini"`) || !strings.Contains(rec.Body.String(), `"has_own_key":false`) {
		t.Errorf("เปลี่ยน JWT_SECRET แล้วถอด key เก่าไม่ได้ ต้องถือว่าไม่มี key ได้ %s", rec.Body.String())
	}
}
