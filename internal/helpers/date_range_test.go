package helpers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/utils"
)

func TestDateRangeParams(t *testing.T) {
	thisMonth := utils.MonthRange(time.Now())

	tests := []struct {
		name   string
		query  string
		want   utils.DateRange
		wantOK bool
	}{
		{"ส่งครบ ได้ช่วงนั้น", "?from=2026-11-01&to=2026-11-16", utils.DateRange{From: "2026-11-01", To: "2026-11-16"}, true},
		{"วันเดียว", "?from=2026-11-16&to=2026-11-16", utils.DateRange{From: "2026-11-16", To: "2026-11-16"}, true},
		{"ไม่ส่ง ได้ทั้งเดือนนี้", "", thisMonth, true},
		{"ส่งแค่ from", "?from=2026-11-01", utils.DateRange{}, false},
		{"from อยู่หลัง to", "?from=2026-11-16&to=2026-11-01", utils.DateRange{}, false},
		{"รูปแบบผิด", "?from=2026-11&to=2026-12", utils.DateRange{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodGet, "/"+tt.query, nil)

			got, ok := DateRangeParams(c)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("ได้ (%+v, %v), ต้องได้ (%+v, %v)", got, ok, tt.want, tt.wantOK)
			}
			if !ok && rec.Body.String() != `{"error":"date_range_invalid"}` {
				t.Errorf("body = %s, ต้องได้ date_range_invalid", rec.Body.String())
			}
		})
	}
}
