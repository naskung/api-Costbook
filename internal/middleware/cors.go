package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS อนุญาตให้เว็บ (Flutter) ที่อยู่คนละ port / โดเมน เรียก API ได้
// ถ้าไม่มี browser จะบล็อก request จากเว็บ
func CORS(allowedOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", allowedOrigin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// browser ส่ง OPTIONS มาถามก่อนว่าอนุญาตไหม → ตอบ 204 แล้วจบ ไม่ต้องไปถึง handler
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
