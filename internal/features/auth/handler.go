package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/helpers"
	"github.com/naskung/api-Costbook/internal/middleware"
	"github.com/naskung/api-Costbook/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

// tokenDuration อายุของ token หลัง login (หมดแล้วต้อง login ใหม่)
const tokenDuration = 7 * 24 * time.Hour

type Handler struct {
	repo      Repository
	jwtSecret string
}

func NewHandler(repo Repository, jwtSecret string) *Handler {
	return &Handler{repo: repo, jwtSecret: jwtSecret}
}

// RegisterRoutes ต่อ route ของ auth เข้ากับกลุ่ม /api/v1
// authRequired = middleware ตรวจ token ใส่เฉพาะ route ที่ต้อง login ก่อน
func (h *Handler) RegisterRoutes(api gin.IRouter, authRequired gin.HandlerFunc) {
	api.POST("/auth/login", h.Login)
	api.GET("/auth/me", authRequired, h.Me)
}

// Login POST /api/v1/auth/login
// อีเมลไม่มีในระบบ หรือรหัสผ่านผิด ตอบ invalid_credentials เหมือนกัน → คนเดาจะไม่รู้ว่าอีเมลนี้มีจริงไหม
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		helpers.WriteError(c, http.StatusBadRequest, "invalid_json")
		return
	}

	code := req.Validate()
	if code != "" {
		helpers.WriteError(c, http.StatusBadRequest, code)
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	user, err := h.repo.FindByEmail(c.Request.Context(), email)
	if errors.Is(err, ErrNotFound) {
		helpers.WriteError(c, http.StatusUnauthorized, "invalid_credentials")
		return
	}
	if err != nil {
		helpers.ServerError(c, err)
		return
	}

	// เทียบรหัสผ่านที่กรอก กับ hash ใน DB
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		helpers.WriteError(c, http.StatusUnauthorized, "invalid_credentials")
		return
	}

	token, err := utils.CreateToken(user.ID, h.jwtSecret, tokenDuration)
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, LoginResponse{Token: token, User: user})
}

// Me GET /api/v1/auth/me (ต้องส่ง token มาด้วย)
func (h *Handler) Me(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)

	user, err := h.repo.FindByID(c.Request.Context(), userID)
	if errors.Is(err, ErrNotFound) {
		// token ถูกต้อง แต่ผู้ใช้ถูกลบไปแล้ว
		helpers.WriteError(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, user)
}
