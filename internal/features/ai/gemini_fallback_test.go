package ai

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/middleware"
)

// slowGemini Gemini ที่ค้าง ไม่ตอบจนกว่าจะหมดเวลา
type slowGemini struct{ calls int }

func (s *slowGemini) Generate(ctx context.Context, req geminiRequest) (geminiResponse, error) {
	s.calls++
	<-ctx.Done()
	return geminiResponse{}, ctx.Err()
}

var (
	geminiBusy  = &GeminiError{Status: http.StatusServiceUnavailable, Message: "This model is currently experiencing high demand"}
	geminiQuota = &GeminiError{Status: http.StatusTooManyRequests, Message: "quota"}
	geminiBad   = &GeminiError{Status: http.StatusBadRequest, Message: "bad request"}
)

func fallbackOf(clients ...Gemini) *fallbackGemini {
	models := []string{}
	for i := range clients {
		models = append(models, "model-"+string(rune('a'+i)))
	}
	return &fallbackGemini{models: models, clients: clients, timeout: time.Second}
}

func TestGeminiModelsOf(t *testing.T) {
	got := geminiModelsOf("gemini-3.6-flash", []string{"gemini-3.5-flash", "", "gemini-3.6-flash", "gemini-3.5-flash-lite"})
	want := "gemini-3.6-flash,gemini-3.5-flash,gemini-3.5-flash-lite"
	if strings.Join(got, ",") != want {
		t.Errorf("รุ่นหลักก่อน ตามด้วยรุ่นสำรองที่ไม่ซ้ำ ได้ %v", got)
	}
}

func TestFallbackGemini_TriesNextModel(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"Google ยุ่ง (503)", geminiBusy},
		{"โควตาของรุ่นนี้เต็ม (429)", geminiQuota},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			primary := &fakeGemini{err: tt.err}
			backup := &fakeGemini{replies: []string{geminiTextReply}}
			resp, err := fallbackOf(primary, backup).Generate(context.Background(), geminiRequest{})
			if err != nil || len(resp.Candidates) != 1 || len(backup.sent) != 1 {
				t.Errorf("รุ่นหลักใช้ไม่ได้ ต้องได้คำตอบจากรุ่นสำรอง ได้ %+v %v", resp, err)
			}
		})
	}
}

func TestFallbackGemini_SlowModelTimesOut(t *testing.T) {
	slow := &slowGemini{}
	backup := &fakeGemini{replies: []string{geminiTextReply}}
	f := fallbackOf(slow, backup)
	f.timeout = 20 * time.Millisecond

	_, err := f.Generate(context.Background(), geminiRequest{})
	if err != nil || slow.calls != 1 || len(backup.sent) != 1 {
		t.Errorf("รุ่นหลักค้างเกินเวลา ต้องไปรุ่นสำรอง ได้ %v", err)
	}
}

func TestFallbackGemini_StaysOnBackup(t *testing.T) {
	primary := &fakeGemini{err: geminiBusy}
	backup := &fakeGemini{replies: []string{geminiListReply, geminiTextReply}}
	f := fallbackOf(primary, backup)
	f.Generate(context.Background(), geminiRequest{})
	f.Generate(context.Background(), geminiRequest{})
	if len(primary.sent) != 1 || len(backup.sent) != 2 {
		t.Errorf("สลับไปรุ่นสำรองแล้วต้องใช้รุ่นนั้นต่อทั้งงาน ได้ รุ่นหลัก %d ครั้ง รุ่นสำรอง %d ครั้ง", len(primary.sent), len(backup.sent))
	}
}

func TestFallbackGemini_DoesNotRetry(t *testing.T) {
	t.Run("คำขอผิด (400) ลองรุ่นไหนก็ไม่ผ่าน ไม่ลองต่อ", func(t *testing.T) {
		backup := &fakeGemini{}
		_, err := fallbackOf(&fakeGemini{err: geminiBad}, backup).Generate(context.Background(), geminiRequest{})
		if !errors.Is(err, geminiBad) || len(backup.sent) != 0 {
			t.Errorf("ได้ %v (รุ่นสำรองถูกเรียก %d ครั้ง)", err, len(backup.sent))
		}
	})
	t.Run("งานทั้งหมดหมดเวลาแล้ว ไม่ลองต่อ", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		backup := &fakeGemini{}
		_, err := fallbackOf(&slowGemini{}, backup).Generate(ctx, geminiRequest{})
		if err == nil || len(backup.sent) != 0 {
			t.Errorf("ได้ %v (รุ่นสำรองถูกเรียก %d ครั้ง)", err, len(backup.sent))
		}
	})
	t.Run("ทุกรุ่นยุ่ง ได้ error ของรุ่นสุดท้าย", func(t *testing.T) {
		_, err := fallbackOf(&fakeGemini{err: geminiQuota}, &fakeGemini{err: geminiBusy}).Generate(context.Background(), geminiRequest{})
		if !errors.Is(err, geminiBusy) {
			t.Errorf("ได้ %v", err)
		}
	})
}

// newModelRouter Gemini ฟรีที่แยกตัวปลอมตามรุ่น (รุ่นที่ไม่ได้เตรียมไว้ = ยุ่ง)
func newModelRouter(cfg Config, byModel map[string]*fakeGemini) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(&fakeRepository{}, fakeShop{}, cfg)
	h.now = func() time.Time { return testToday }
	h.newGemini = func(apiKey, model string) Gemini {
		if fake, ok := byModel[model]; ok {
			return fake
		}
		return &fakeGemini{err: geminiBusy}
	}
	h.RegisterRoutes(r.Group("/api/v1"), middleware.Auth(testSecret))
	return r
}

func TestHandler_Chat_FallsBackToBackupModel(t *testing.T) {
	backup := &fakeGemini{replies: []string{geminiTextReply}}
	cfg := Config{GeminiKey: "free-key", GeminiModel: "gemini-busy", GeminiFallbackModels: []string{"gemini-backup"}, Secret: testSecret}
	rec := sendRequest(newModelRouter(cfg, map[string]*fakeGemini{"gemini-backup": backup}), http.MethodPost, "/api/v1/ai/chat", askBody)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "ยอดขายเดือนนี้") {
		t.Errorf("รุ่นหลักยุ่ง ต้องได้คำตอบจากรุ่นสำรอง ได้ %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Chat_DefaultFallbacks(t *testing.T) {
	backup := &fakeGemini{replies: []string{geminiTextReply}}
	cfg := Config{GeminiKey: "free-key", GeminiModel: "gemini-busy", Secret: testSecret}
	rec := sendRequest(newModelRouter(cfg, map[string]*fakeGemini{DefaultGeminiFallbacks[1]: backup}), http.MethodPost, "/api/v1/ai/chat", askBody)

	if rec.Code != http.StatusOK {
		t.Errorf("ไม่ตั้ง GEMINI_FALLBACK_MODELS ต้องใช้รุ่นสำรองค่าเริ่มต้น ได้ %d %s", rec.Code, rec.Body.String())
	}
}
