package helpers

import (
	"time"

	"github.com/gin-gonic/gin"
)

// MonthStart อ่าน ?month=2026-10 จาก URL แล้วคืนวันแรกของเดือน "2026-10-01" ไว้ใช้ใน SQL
// ไม่ส่ง month มา → ใช้เดือนปัจจุบัน / รูปแบบผิด → ok = false (handler ตอบ 400 month_invalid)
func MonthStart(c *gin.Context) (string, bool) {
	month := c.Query("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}

	start, err := time.Parse("2006-01", month)
	if err != nil {
		return "", false
	}
	return start.Format("2006-01-02"), true
}
