package ai

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"strings"
)

// ผู้ให้บริการ AI ที่ใช้อยู่
// openai = AI เจ้าไหนก็ได้ที่ใช้รูปแบบ API ของ OpenAI (ร้านตั้ง ที่อยู่ API + key + รุ่น เอง)
const (
	providerClaude = "claude"
	providerGemini = "gemini"
	providerOpenAI = "openai"
	providerNone   = "none"
)

// maxModelLength ชื่อรุ่นยาวไม่เกินเท่านี้
const maxModelLength = 200

// ที่มาของ key: settings = ร้านใส่เองในหน้าตั้งค่า, env = ไฟล์ .env ของ API
const (
	sourceSettings = "settings"
	sourceEnv      = "env"
)

// ErrNoWebSearch AI ที่ใช้อยู่ค้นเว็บไม่ได้ (Gemini ที่ไม่มี TAVILY_API_KEY และไม่ได้เปิด GEMINI_WEB_SEARCH) → เช็คราคาตลาดไม่ได้
var ErrNoWebSearch = errors.New("current AI cannot search the web")

// Config ค่าของ AI จาก .env
// ClaudeKey / GeminiKey ไม่ใส่ก็ได้ / Secret ไว้เข้ารหัส key ที่ร้านใส่ในหน้าตั้งค่า (ใช้ JWT_SECRET)
// GeminiWebSearch = เปิดให้ Gemini ค้น Google (ต้องเป็น Gemini แบบเสียเงิน แบบฟรีค้นไม่ได้)
// TavilyKey = ตัวค้นเว็บฟรี ให้ Gemini ฟรีเช็คราคาตลาดได้
type Config struct {
	ClaudeKey         string
	GeminiKey         string
	GeminiModel       string
	GeminiWebSearch   bool
	GeminiSearchModel string
	TavilyKey         string
	// GeminiFallbackModels รุ่นสำรองเมื่อรุ่นหลักยุ่ง / โควตาเต็ม (ว่าง = DefaultGeminiFallbacks)
	GeminiFallbackModels []string
	Secret               string
}

// Assistant AI ที่ใช้ตอบ (Claude / Gemini / AI แบบ OpenAI) — handler ไม่ต้องรู้ว่าเป็นเจ้าไหน
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

// geminiAssistant gemini = รุ่นที่แชท / จัดผล
// เช็คราคาตลาด: useGoogle (GEMINI_WEB_SEARCH) → search ค้น Google / ไม่งั้นมี searcher (Tavily) → ค้นเว็บเอง / ไม่มีทั้งคู่ → ค้นไม่ได้
type geminiAssistant struct {
	gemini    Gemini
	search    Gemini
	useGoogle bool
	searcher  WebSearcher
}

func (a geminiAssistant) Chat(ctx context.Context, tools toolbox, history []ChatMessage) (ChatReply, error) {
	return geminiChat(ctx, a.gemini, tools, history)
}

func (a geminiAssistant) Research(ctx context.Context, prompt string) (MarketReport, error) {
	if a.useGoogle {
		return geminiResearch(ctx, a.search, a.gemini, prompt)
	}
	if a.searcher != nil {
		return geminiSearchResearch(ctx, a.searcher, a.gemini, prompt)
	}
	return MarketReport{}, ErrNoWebSearch
}

// openAIAssistant AI แบบ OpenAI — เช็คราคาตลาดค้นเว็บผ่าน Tavily แบบเดียวกับ Gemini ฟรี (ไม่มี TAVILY_API_KEY = ค้นไม่ได้)
type openAIAssistant struct {
	client   OpenAI
	searcher WebSearcher
}

func (a openAIAssistant) Chat(ctx context.Context, tools toolbox, history []ChatMessage) (ChatReply, error) {
	return openAIChat(ctx, a.client, tools, history)
}

func (a openAIAssistant) Research(ctx context.Context, prompt string) (MarketReport, error) {
	if a.searcher == nil {
		return MarketReport{}, ErrNoWebSearch
	}
	return searchResearch(ctx, a.searcher, openAIAsker(a.client), prompt)
}

// aiSettings AI ที่ร้านตั้งไว้ (key ถอดรหัสแล้ว) — Provider ว่าง = ยังไม่ได้ตั้ง
type aiSettings struct {
	Provider string
	BaseURL  string
	Model    string
	APIKey   string
}

// Status AI ที่ใช้อยู่ตอนนี้ (ตอบ Flutter ทาง GET /ai/status) — ไม่ส่ง key ตัวจริงกลับไปเด็ดขาด
// key_hint = 4 ตัวท้ายของ key ที่ร้านใส่ไว้ เช่น "…a1B2" / base_url = ที่อยู่ API (เฉพาะ AI แบบ OpenAI)
type Status struct {
	Enabled       bool   `json:"enabled"`
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	BaseURL       string `json:"base_url"`
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

// pickProvider เลือก AI: ที่ร้านตั้งในหน้าตั้งค่า (Claude / แบบ OpenAI) → key Claude ใน .env → Gemini ฟรี → ไม่มี
func pickProvider(saved aiSettings, cfg Config) Status {
	status := Status{
		Provider: providerNone, FreeAvailable: cfg.GeminiKey != "",
		HasOwnKey: saved.Provider != "", KeyHint: keyHint(saved.APIKey),
	}
	switch {
	case saved.Provider != "":
		status.Provider, status.Source = saved.Provider, sourceSettings
	case cfg.ClaudeKey != "":
		status.Provider, status.Source = providerClaude, sourceEnv
	case cfg.GeminiKey != "":
		status.Provider, status.Source = providerGemini, sourceEnv
	}
	switch status.Provider {
	case providerClaude:
		status.Enabled, status.Model, status.WebSearch = true, model, true
	case providerOpenAI:
		status.Enabled, status.Model, status.BaseURL, status.WebSearch = true, saved.Model, saved.BaseURL, cfg.TavilyKey != ""
	case providerGemini:
		status.Enabled, status.Model, status.WebSearch = true, geminiModelOf(cfg), cfg.GeminiWebSearch || cfg.TavilyKey != ""
	}
	return status
}

// SaveAIRequest ร้านเลือก AI เอง (PUT /ai/key)
// provider ไม่ส่ง = claude (แบบเดิม: ส่งแค่ api_key ของ Claude) / openai ต้องมี base_url + model (api_key ว่างได้)
type SaveAIRequest struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	BaseURL  string `json:"base_url"`
	Model    string `json:"model"`
}

// normalized ตัดช่องว่าง / provider ว่าง = claude / ตัด / ท้ายที่อยู่ API (ใส่ .../v1/ หรือ .../v1 ได้เหมือนกัน)
func (r SaveAIRequest) normalized() SaveAIRequest {
	r.Provider = strings.TrimSpace(r.Provider)
	if r.Provider == "" {
		r.Provider = providerClaude
	}
	r.APIKey = strings.TrimSpace(r.APIKey)
	r.BaseURL = strings.TrimRight(strings.TrimSpace(r.BaseURL), "/")
	r.Model = strings.TrimSpace(r.Model)
	return r
}

// Validate ตรวจก่อนลองต่อ AI จริง คืน error code ("" = ผ่าน)
func (r SaveAIRequest) Validate() string {
	if !slices.Contains([]string{providerClaude, providerOpenAI}, r.Provider) {
		return "ai_provider_invalid"
	}
	if r.Provider == providerClaude {
		if !validClaudeKey(r.APIKey) {
			return "ai_key_format_invalid"
		}
		return ""
	}
	if !validBaseURL(r.BaseURL) {
		return "ai_base_url_invalid"
	}
	if r.Model == "" || len(r.Model) > maxModelLength || strings.ContainsAny(r.Model, " \t\n") {
		return "ai_model_required"
	}
	if strings.ContainsAny(r.APIKey, " \t\n") {
		return "ai_key_format_invalid"
	}
	return ""
}

// validBaseURL ที่อยู่ API ต้องเป็น http(s)://โดเมน... เช่น https://api.deepseek.com/v1 หรือ http://localhost:11434/v1
func validBaseURL(text string) bool {
	parsed, err := url.Parse(text)
	if err != nil {
		return false
	}
	return (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Host != "" && parsed.RawQuery == ""
}

// geminiSearchModelOf รุ่นที่ใช้ค้น Google — ไม่ตั้งไว้ใช้รุ่นเดียวกับแชท
func geminiSearchModelOf(cfg Config) string {
	if cfg.GeminiSearchModel == "" {
		return geminiModelOf(cfg)
	}
	return cfg.GeminiSearchModel
}

func geminiFallbacksOf(cfg Config) []string {
	if len(cfg.GeminiFallbackModels) == 0 {
		return DefaultGeminiFallbacks
	}
	return cfg.GeminiFallbackModels
}

func geminiModelOf(cfg Config) string {
	if cfg.GeminiModel == "" {
		return DefaultGeminiModel
	}
	return cfg.GeminiModel
}
