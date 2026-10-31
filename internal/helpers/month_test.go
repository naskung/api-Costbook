package helpers

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestMonthStart(t *testing.T) {
	currentMonth := time.Now().Format("2006-01") + "-01"

	tests := []struct {
		name   string
		url    string
		want   string
		wantOK bool
	}{
		{"month=2026-10 ได้ 2026-10-01", "/?month=2026-10", "2026-10-01", true},
		{"ไม่ส่ง month ได้วันแรกของเดือนนี้", "/", currentMonth, true},
		{"month=2026-13 ไม่มีเดือนนี้", "/?month=2026-13", "", false},
		{"month=10-2026 รูปแบบผิด", "/?month=10-2026", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", tt.url, nil)

			got, ok := MonthStart(c)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("MonthStart(%s) = (%q, %v), ต้องได้ (%q, %v)", tt.url, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
