package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/helpers"
	"github.com/naskung/api-Costbook/internal/utils"
)

// UserIDKey ชื่อที่ใช้เก็บ id ผู้ใช้ไว้ใน gin.Context หลังตรวจ token ผ่าน
// handler อ่านด้วย c.GetString(middleware.UserIDKey)
const UserIDKey = "user_id"

// Auth ตรวจ token จาก header "Authorization: Bearer <token>"
// ไม่มี token / token ผิด / หมดอายุ → ตอบ 401 unauthorized ไม่ส่งต่อไปที่ handler
func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, found := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !found {
			helpers.WriteError(c, http.StatusUnauthorized, "unauthorized")
			return
		}

		userID, err := utils.ParseToken(token, jwtSecret)
		if err != nil {
			helpers.WriteError(c, http.StatusUnauthorized, "unauthorized")
			return
		}

		c.Set(UserIDKey, userID)
		c.Next()
	}
}
