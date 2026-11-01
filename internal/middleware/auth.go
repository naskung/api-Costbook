package middleware

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/helpers"
	"github.com/naskung/api-Costbook/internal/utils"
)

// UserIDKey ชื่อที่ใช้เก็บ id ผู้ใช้ไว้ใน gin.Context หลังตรวจ token ผ่าน
// handler อ่านด้วย c.GetString(middleware.UserIDKey)
const UserIDKey = "user_id"

// RoleKey ชื่อที่ใช้เก็บ role ของผู้ใช้ไว้ใน gin.Context (มีเมื่อใช้ Authenticated) — c.GetString(middleware.RoleKey)
const RoleKey = "user_role"

// Auth ตรวจ token จาก header "Authorization: Bearer <token>"
// ไม่มี token / token ผิด / หมดอายุ → ตอบ 401 unauthorized ไม่ส่งต่อไปที่ handler
func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !verifyToken(c, jwtSecret) {
			return
		}
		c.Next()
	}
}

// UserLookup อ่าน role + สถานะของผู้ใช้จาก DB (ไม่เจอผู้ใช้ = ไม่ active)
type UserLookup func(ctx context.Context, userID string) (role string, active bool, err error)

// Authenticated ตรวจ token แบบ Auth + เช็คผู้ใช้ใน DB ทุก request:
// บัญชีถูกปิด / ถูกลบ → 401 ทันที (ไม่ต้องรอ token หมดอายุ) และเก็บ role ไว้ใน RoleKey ให้ RequireRole ใช้
func Authenticated(jwtSecret string, lookup UserLookup) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !verifyToken(c, jwtSecret) {
			return
		}
		role, active, err := lookup(c.Request.Context(), c.GetString(UserIDKey))
		if err != nil {
			helpers.ServerError(c, err)
			return
		}
		if !active {
			helpers.WriteError(c, http.StatusUnauthorized, "unauthorized")
			return
		}
		c.Set(RoleKey, role)
		c.Next()
	}
}

// RequireRole ให้ผ่านเฉพาะ role ที่กำหนด ไม่ใช่ → 403 forbidden (ใช้ต่อจาก Authenticated)
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !slices.Contains(roles, c.GetString(RoleKey)) {
			helpers.WriteError(c, http.StatusForbidden, "forbidden")
			return
		}
		c.Next()
	}
}

// verifyToken อ่าน + ตรวจ token แล้วเก็บ id ผู้ใช้ไว้ใน UserIDKey / ไม่ผ่าน → ตอบ 401 แล้วคืน false
func verifyToken(c *gin.Context, jwtSecret string) bool {
	token, found := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
	if !found {
		helpers.WriteError(c, http.StatusUnauthorized, "unauthorized")
		return false
	}
	userID, err := utils.ParseToken(token, jwtSecret)
	if err != nil {
		helpers.WriteError(c, http.StatusUnauthorized, "unauthorized")
		return false
	}
	c.Set(UserIDKey, userID)
	return true
}
