package helpers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/utils"
)

// DateRangeParams อ่านช่วงวันที่ ?from=2026-11-01&to=2026-11-16 จาก URL (นับรวมทั้ง 2 วัน)
// ไม่ส่งทั้งคู่ → ทั้งเดือนนี้ / วันนี้วันเดียว ส่ง from = to = วันนี้
// ส่งไม่ครบ / รูปแบบผิด / from อยู่หลัง to / ยาวเกิน 366 วัน → ตอบ 400 date_range_invalid แล้วคืน ok = false (handler return ได้เลย)
func DateRangeParams(c *gin.Context) (utils.DateRange, bool) {
	from := c.Query("from")
	to := c.Query("to")
	if from == "" && to == "" {
		return utils.MonthRange(time.Now()), true
	}
	period, ok := utils.NewDateRange(from, to)
	if !ok {
		WriteError(c, http.StatusBadRequest, "date_range_invalid")
	}
	return period, ok
}
