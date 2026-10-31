package helpers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// WriteError ตอบ error เป็น {"error": "code"} แล้วหยุดไม่ให้ทำ handler ถัดไป
// ใช้ได้ทั้งใน handler และ middleware
func WriteError(c *gin.Context, status int, code string) {
	c.AbortWithStatusJSON(status, gin.H{"error": code})
}

// ServerError เขียน error จริงลง log แล้วตอบ client แค่ "server_error"
// (ไม่ส่งรายละเอียด error ให้ client เพราะอาจมีข้อมูลภายในหลุดออกไป)
func ServerError(c *gin.Context, err error) {
	slog.Error("server error", "method", c.Request.Method, "path", c.Request.URL.Path, "error", err)
	WriteError(c, http.StatusInternalServerError, "server_error")
}
