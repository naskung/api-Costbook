package ai

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"
	"github.com/anthropics/anthropic-sdk-go/v2/shared/constant"
	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/helpers"
)

// model รุ่นของ Claude ที่ใช้ทั้งเช็คราคาตลาดและแชท
const model = "claude-opus-5-5"

// error ของงาน AI (handler แปลงเป็น error code ให้ Flutter)
var (
	ErrRefused      = errors.New("claude refused the request")
	ErrNoReport     = errors.New("claude did not send a market report")
	ErrTooManySteps = errors.New("claude used too many steps")
)

// Claude ส่งคำขอไปหา Claude 1 ครั้ง
// แยกเป็น interface เพื่อให้ test ใช้ตัวปลอมได้ (ไม่ต้องเรียก API จริง ไม่เสียเงิน)
type Claude interface {
	Send(ctx context.Context, params anthropic.BetaMessageNewParams) (*anthropic.BetaMessage, error)
	CheckKey(ctx context.Context) error
}

type claudeClient struct {
	client anthropic.Client
}

// NewClaude สร้างตัวเรียก Claude — ไม่มี API key = ปิด AI (คืน nil, handler ตอบ ai_not_configured)
func NewClaude(apiKey string) Claude {
	if apiKey == "" {
		return nil
	}
	return &claudeClient{client: anthropic.NewClient(option.WithAPIKey(apiKey))}
}

// Send ใส่รุ่นที่ใช้ + เปิด fallback (Claude ปฏิเสธคำขอเพราะนโยบายความปลอดภัย → ให้รุ่นอื่นตอบแทนในคำขอเดียวกัน)
func (c *claudeClient) Send(ctx context.Context, params anthropic.BetaMessageNewParams) (*anthropic.BetaMessage, error) {
	params.Model = model
	params.Betas = append(params.Betas, anthropic.AnthropicBetaServerSideFallback2026_07_01)
	params.Fallbacks = anthropic.BetaFallbacksParamUnion{OfDefault: constant.ValueOf[constant.Default]()}
	return c.client.Beta.Messages.New(ctx, params)
}

// CheckKey ลองใช้ key ดู (ขอรายชื่อรุ่น 1 ตัว ไม่เสียค่า AI) key ผิดได้ *anthropic.Error 401
func (c *claudeClient) CheckKey(ctx context.Context) error {
	_, err := c.client.Models.List(ctx, anthropic.ModelListParams{Limit: anthropic.Int(1)})
	return err
}

// writeAIError แปลง error ของงาน AI เป็น error code ที่ Flutter แปลเป็นข้อความได้ + เขียนสาเหตุลง log ทุกครั้ง
func writeAIError(c *gin.Context, err error) {
	var apiErr *anthropic.Error
	var geminiErr *GeminiError
	var tavilyErr *TavilyError
	switch {
	case errors.Is(err, ErrNoWebSearch):
		writeAICode(c, http.StatusConflict, "ai_search_unavailable", err.Error())
	case errors.As(err, &geminiErr):
		writeGeminiError(c, geminiErr)
	case errors.As(err, &tavilyErr):
		writeTavilyError(c, tavilyErr)
	case errors.Is(err, ErrRefused):
		writeAICode(c, http.StatusUnprocessableEntity, "ai_refused", err.Error())
	case errors.Is(err, ErrNoReport), errors.Is(err, ErrTooManySteps):
		writeAICode(c, http.StatusBadGateway, "ai_no_result", err.Error())
	case errors.Is(err, context.DeadlineExceeded):
		writeAICode(c, http.StatusGatewayTimeout, "ai_timeout", err.Error())
	case errors.As(err, &apiErr):
		writeClaudeAPIError(c, apiErr)
	default:
		helpers.ServerError(c, err)
	}
}

// writeAICode ตอบ error code ให้ Flutter + เขียนสาเหตุลง log (ดูได้ในหน้าต่าง make run ว่า AI ใช้ไม่ได้เพราะอะไร)
func writeAICode(c *gin.Context, status int, code, detail string) {
	slog.Warn("AI ใช้ไม่ได้", "code", code, "detail", detail)
	helpers.WriteError(c, status, code)
}

// writeGeminiError error ที่ API ของ Gemini ตอบกลับมา
// key ผิด (400 ที่บอกว่า API key / 401 / 403) = ai_key_invalid, 429 = โควตาฟรีหมด, 5xx = ไม่ว่าง
func writeGeminiError(c *gin.Context, err *GeminiError) {
	isKeyError := err.Status == http.StatusUnauthorized || err.Status == http.StatusForbidden ||
		(err.Status == http.StatusBadRequest && strings.Contains(err.Message, "API key"))
	switch {
	case isKeyError:
		writeAICode(c, http.StatusServiceUnavailable, "ai_key_invalid", err.Error())
	case err.Status == http.StatusTooManyRequests:
		writeAICode(c, http.StatusServiceUnavailable, "ai_quota_exceeded", err.Error())
	case err.Status >= 500:
		writeAICode(c, http.StatusServiceUnavailable, "ai_busy", err.Error())
	default:
		slog.Error("เรียก Gemini ไม่สำเร็จ", "status", err.Status, "error", err.Message)
		helpers.WriteError(c, http.StatusBadGateway, "ai_error")
	}
}

// writeClaudeAPIError error ที่ API ของ Claude ตอบกลับมา
// เครดิตหมด = ai_credit_low, 401 / 403 = API key ผิด, 429 / 5xx = Claude ไม่ว่าง (ลองใหม่ทีหลัง), อื่น ๆ = เขียนลง log แล้วตอบ ai_error
func writeClaudeAPIError(c *gin.Context, apiErr *anthropic.Error) {
	detail := apiErr.RawJSON()
	switch {
	case isCreditError(apiErr):
		writeAICode(c, http.StatusServiceUnavailable, "ai_credit_low", detail)
	case apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden:
		writeAICode(c, http.StatusServiceUnavailable, "ai_key_invalid", detail)
	case apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode >= 500:
		writeAICode(c, http.StatusServiceUnavailable, "ai_busy", detail)
	default:
		slog.Error("เรียก Claude ไม่สำเร็จ", "status", apiErr.StatusCode, "error", detail)
		helpers.WriteError(c, http.StatusBadGateway, "ai_error")
	}
}

// isCreditError เครดิตของ key หมด — Anthropic ตอบ 400 ข้อความ "Your credit balance is too low ..." หรือ error แบบ billing_error
// (ตอนบันทึก key ลองด้วยการขอรายชื่อรุ่นซึ่งไม่เสียเงิน จึงยังรู้ไม่ได้ว่าเครดิตหมด จะรู้ตอนเรียกใช้จริง)
func isCreditError(apiErr *anthropic.Error) bool {
	return apiErr.Type() == anthropic.ErrorTypeBillingError || strings.Contains(apiErr.RawJSON(), "credit balance")
}

// writeTavilyError error ของตัวค้นเว็บฟรี (Tavily) — 401 / 403 = key ผิด, 429 / 432 / 433 = ค้นถี่เกิน / เครดิตเดือนนี้หมด
func writeTavilyError(c *gin.Context, err *TavilyError) {
	switch err.Status {
	case http.StatusUnauthorized, http.StatusForbidden:
		writeAICode(c, http.StatusServiceUnavailable, "search_key_invalid", err.Error())
	case http.StatusTooManyRequests, tavilyPlanLimit, tavilyPayLimit:
		writeAICode(c, http.StatusServiceUnavailable, "search_quota_exceeded", err.Error())
	default:
		slog.Error("ค้นเว็บ (Tavily) ไม่สำเร็จ", "status", err.Status, "error", err.Message)
		helpers.WriteError(c, http.StatusBadGateway, "ai_error")
	}
}
