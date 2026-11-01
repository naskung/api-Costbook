package helpers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// maxPageLimit ขอได้มากสุดกี่รายการต่อครั้ง (กันขอทีเดียวเยอะเกินจน DB / เน็ตช้า)
const maxPageLimit = 200

// PageParams อ่าน ?limit= (เอากี่รายการ) และ ?offset= (ข้ามไปกี่รายการ) ของรายการแบบโหลดทีละหน้า
// ไม่ส่ง = 0 → limit 0 คือเอาทั้งหมด (หน้าที่ต้องใช้ข้อมูลครบ เช่น หน้าขาย / AI ยังเรียกแบบเดิมได้)
// ไม่ใช่ตัวเลข / ติดลบ / limit เกิน maxPageLimit → ตอบ 400 page_invalid แล้วคืน ok = false (handler return ได้เลย)
// เช่น ?limit=20&offset=40 = รายการที่ 41–60
func PageParams(c *gin.Context) (limit, offset int, ok bool) {
	limit, isLimitValid := queryInt(c, "limit")
	offset, isOffsetValid := queryInt(c, "offset")
	if !isLimitValid || !isOffsetValid || limit > maxPageLimit {
		WriteError(c, http.StatusBadRequest, "page_invalid")
		return 0, 0, false
	}
	return limit, offset, true
}

// queryInt อ่านตัวเลข 0 ขึ้นไปจาก query string (ไม่ส่ง = 0)
func queryInt(c *gin.Context, name string) (int, bool) {
	text := c.Query(name)
	if text == "" {
		return 0, true
	}
	number, err := strconv.Atoi(text)
	return number, err == nil && number >= 0
}
