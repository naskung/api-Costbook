package ai

import (
	"context"
	"errors"
	"strings"
)

// ผู้ให้บริการ AI ที่ใช้อยู่
const (
	providerClaude = "claude"
	providerGemini = "gemini"
	providerNone   = "none"
)

// ที่มาของ key: settings = ร้านใส่เองในหน้าตั้งค่า, env = ไฟล์ .env ของ API
const (
	sourceSettings = "settings"
	sourceEnv      = "env"
)

// ErrNoWebSearch AI ที่ใช้อยู่ค้นเว็บไม่ได้ (Gemini ฟรี) → เช็คราคาตลาดไม่ได้ ต้องใส่ key Claude
var ErrNoWebSearch = errors.New("current AI cannot search the web")

// Config ค่าของ AI จาก .env
// ClaudeKey / GeminiKey ไม่ใส่ก็ได้ / Secret ไว้เข้ารหัส key ที่ร้านใส่ในหน้าตั้งค่า (ใช้ JWT_SECRET)
type Config struct {
	ClaudeKey   string
	GeminiKey   string
	GeminiModel string
	Secret      string
}

// Assistant AI ที่ใช้ตอบ (Claude หรือ Gemini) — handler ไม่ต้องรู้ว่าเป็นเจ้าไหน
type Assistant interface {
	Chat(ctx context.Context, tools toolbox, history []ChatMessage) (ChatReply, error)
	Research(ctx context.Context, prompt string) (MarketReport, error)
}

type claudeAssistant struct {
	claude Claude
}

func (a claudeAssistant) Chat(ctx context.Context, tools toolbox, history []ChatMessage) (ChatReply, error) {
	return chat(ctx, a.claude, tools, history)
}

func (a claudeAssistant) Research(ctx context.Context, prompt string) (MarketReport, error) {
	return research(ctx, a.claude, prompt)
}

type geminiAssistant struct {
	gemini Gemini
}

func (a geminiAssistant) Chat(ctx context.Context, tools toolbox, history []ChatMessage) (ChatReply, error) {
	return geminiChat(ctx, a.gemini, tools, history)
}

// Research Gemini ฟรีค้นเว็บไม่ได้ (Google Search ไม่อยู่ในโควตาฟรี) → ไม่เดาราคาเอง
func (a geminiAssistant) Research(ctx context.Context, prompt string) (MarketReport, error) {
	return MarketReport{}, ErrNoWebSearch
}

// Status AI ที่ใช้อยู่ตอนนี้ (ตอบ Flutter ทาง GET /ai/status) — ไม่ส่ง key ตัวจริงกลับไปเด็ดขาด
// key_hint = 4 ตัวท้ายของ key ที่ร้านใส่ไว้ เช่น "…a1B2"
type Status struct {
	Enabled       bool   `json:"enabled"`
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	Source        string `json:"source"`
	WebSearch     bool   `json:"web_search"`
	HasOwnKey     bool   `json:"has_own_key"`
	KeyHint       string `json:"key_hint"`
	FreeAvailable bool   `json:"free_available"`
}

// keyHint 4 ตัวท้ายของ key ไว้ให้ผู้ใช้จำได้ว่าใส่ key ไหนไว้
func keyHint(key string) string {
	if len(key) < 8 {
		return ""
	}
	return "…" + key[len(key)-4:]
}

// validClaudeKey key ของ Claude ขึ้นต้นด้วย sk-ant- และยาวพอ (เช็คก่อนเรียก API จริง)
func validClaudeKey(key string) bool {
	return strings.HasPrefix(key, "sk-ant-") && len(key) >= 20 && !strings.ContainsAny(key, " \t\n")
}

// pickProvider เลือก AI: key ที่ร้านใส่ในหน้าตั้งค่า → key Claude ใน .env → Gemini ฟรี → ไม่มี
func pickProvider(savedKey string, cfg Config) Status {
	status := Status{Provider: providerNone, FreeAvailable: cfg.GeminiKey != "", HasOwnKey: savedKey != "", KeyHint: keyHint(savedKey)}
	switch {
	case savedKey != "":
		status.Provider, status.Source = providerClaude, sourceSettings
	case cfg.ClaudeKey != "":
		status.Provider, status.Source = providerClaude, sourceEnv
	case cfg.GeminiKey != "":
		status.Provider, status.Source = providerGemini, sourceEnv
	}
	switch status.Provider {
	case providerClaude:
		status.Enabled, status.Model, status.WebSearch = true, model, true
	case providerGemini:
		status.Enabled, status.Model = true, geminiModelOf(cfg)
	}
	return status
}

func geminiModelOf(cfg Config) string {
	if cfg.GeminiModel == "" {
		return DefaultGeminiModel
	}
	return cfg.GeminiModel
}
