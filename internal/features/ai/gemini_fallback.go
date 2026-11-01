package ai

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// DefaultGeminiFallbacks รุ่นสำรองเมื่อรุ่นหลักใช้ไม่ได้ชั่วคราว (เปลี่ยนได้ด้วย GEMINI_FALLBACK_MODELS ใน .env คั่นด้วย ,)
// Gemini ฟรีนับโควตาแยกแต่ละรุ่น → รุ่นหลักเต็ม / ยุ่ง รุ่นอื่นมักยังใช้ได้
var DefaultGeminiFallbacks = []string{"gemini-3.5-flash", "gemini-3.5-flash-lite"}

// geminiAttemptTimeout รอแต่ละรุ่นนานสุดเท่านี้ ไม่ตอบ → ลองรุ่นถัดไป (เคยเจอรุ่นที่คนใช้เยอะค้างเกิน 2 นาที)
const geminiAttemptTimeout = 60 * time.Second

// geminiModelsOf รุ่นหลักก่อน ตามด้วยรุ่นสำรอง (ไม่เอารุ่นซ้ำ / ชื่อว่าง)
func geminiModelsOf(primary string, fallbacks []string) []string {
	models := []string{primary}
	seen := map[string]bool{primary: true}
	for _, model := range fallbacks {
		if model == "" || seen[model] {
			continue
		}
		seen[model] = true
		models = append(models, model)
	}
	return models
}

// retryableGemini error ที่ลองรุ่นอื่นแล้วอาจผ่าน: Google ยุ่ง (5xx), โควตาของรุ่นนี้เต็ม (429), รอนานเกิน
// key ผิด / คำขอผิด (4xx อื่น ๆ) ลองรุ่นไหนก็ไม่ผ่าน → ไม่ลองต่อ
func retryableGemini(err error) bool {
	var geminiErr *GeminiError
	if errors.As(err, &geminiErr) {
		return geminiErr.Status == http.StatusTooManyRequests || geminiErr.Status >= http.StatusInternalServerError
	}
	return errors.Is(err, context.DeadlineExceeded)
}

// fallbackGemini เรียก Gemini ทีละรุ่นตามลำดับจนกว่าจะได้คำตอบ
// สลับไปรุ่นสำรองแล้วใช้รุ่นนั้นต่อทั้งงาน (แชทหลายรอบอยู่กับรุ่นเดียว — คำตอบของ Gemini ผูกกับรุ่นที่ตอบ)
type fallbackGemini struct {
	models  []string
	clients []Gemini
	timeout time.Duration
	start   int
}

func newFallbackGemini(models []string, newGemini func(model string) Gemini) *fallbackGemini {
	clients := []Gemini{}
	for _, model := range models {
		clients = append(clients, newGemini(model))
	}
	return &fallbackGemini{models: models, clients: clients, timeout: geminiAttemptTimeout}
}

func (f *fallbackGemini) Generate(ctx context.Context, req geminiRequest) (geminiResponse, error) {
	var lastErr error
	for i := f.start; i < len(f.clients); i++ {
		attempt, cancel := context.WithTimeout(ctx, f.timeout)
		resp, err := f.clients[i].Generate(attempt, req)
		cancel()
		if err == nil {
			f.start = i
			return resp, nil
		}
		if ctx.Err() != nil || !retryableGemini(err) {
			return geminiResponse{}, err
		}
		if i+1 < len(f.clients) {
			slog.Warn("Gemini รุ่นนี้ใช้ไม่ได้ชั่วคราว ลองรุ่นสำรอง", "model", f.models[i], "next", f.models[i+1], "error", err)
		}
		lastErr = err
	}
	return geminiResponse{}, lastErr
}
