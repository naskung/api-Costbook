package helpers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPageParams(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantLimit  int
		wantOffset int
		wantOK     bool
	}{
		{"ไม่ส่ง = เอาทั้งหมด", "", 0, 0, true},
		{"หน้า 3 (ทีละ 20)", "?limit=20&offset=40", 20, 40, true},
		{"ส่งแค่ limit", "?limit=20", 20, 0, true},
		{"limit เต็มที่ 200 ได้", "?limit=200", 200, 0, true},
		{"limit เกิน 200", "?limit=201", 0, 0, false},
		{"ติดลบ", "?offset=-1", 0, 0, false},
		{"ไม่ใช่ตัวเลข", "?limit=abc", 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodGet, "/x"+tt.query, nil)

			limit, offset, ok := PageParams(c)
			if limit != tt.wantLimit || offset != tt.wantOffset || ok != tt.wantOK {
				t.Errorf("ได้ (%d, %d, %v) ต้องได้ (%d, %d, %v)", limit, offset, ok, tt.wantLimit, tt.wantOffset, tt.wantOK)
			}
			if !ok && (rec.Code != http.StatusBadRequest || rec.Body.String() != `{"error":"page_invalid"}`) {
				t.Errorf("ผิดต้องตอบ 400 page_invalid ได้ %d %s", rec.Code, rec.Body.String())
			}
		})
	}
}
