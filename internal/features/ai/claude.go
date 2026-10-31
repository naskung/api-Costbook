package ai

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

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

// writeAIError แปลง error ของงาน AI เป็น error code ที่ Flutter แปลเป็นข้อความได้
func writeAIError(c *gin.Context, err error) {
	var apiErr *anthropic.Error
	switch {
	case errors.Is(err, ErrRefused):
		helpers.WriteError(c, http.StatusUnprocessableEntity, "ai_refused")
	case errors.Is(err, ErrNoReport), errors.Is(err, ErrTooManySteps):
		slog.Warn("AI ตอบไม่ครบ", "error", err)
		helpers.WriteError(c, http.StatusBadGateway, "ai_no_result")
	case errors.Is(err, context.DeadlineExceeded):
		helpers.WriteError(c, http.StatusGatewayTimeout, "ai_timeout")
	case errors.As(err, &apiErr):
		writeClaudeAPIError(c, apiErr)
	default:
		helpers.ServerError(c, err)
	}
}

// writeClaudeAPIError error ที่ API ของ Claude ตอบกลับมา
// 401 / 403 = API key ผิด, 429 / 5xx = Claude ไม่ว่าง (ลองใหม่ทีหลัง), อื่น ๆ = เขียนลง log แล้วตอบ ai_error
func writeClaudeAPIError(c *gin.Context, apiErr *anthropic.Error) {
	switch {
	case apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden:
		helpers.WriteError(c, http.StatusServiceUnavailable, "ai_key_invalid")
	case apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode >= 500:
		helpers.WriteError(c, http.StatusServiceUnavailable, "ai_busy")
	default:
		slog.Error("เรียก Claude ไม่สำเร็จ", "status", apiErr.StatusCode, "error", apiErr)
		helpers.WriteError(c, http.StatusBadGateway, "ai_error")
	}
}
