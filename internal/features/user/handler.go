package user

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/features/auth"
	"github.com/naskung/api-Costbook/internal/helpers"
	"github.com/naskung/api-Costbook/internal/middleware"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	repo     Repository
	hashCost int
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo, hashCost: bcrypt.DefaultCost}
}

// RegisterRoutes ต่อ route ของโปรไฟล์ (ทุกคน) + จัดการผู้ใช้ (เฉพาะเจ้าของร้าน)
// authRequired ต้องเป็น middleware.Authenticated (มี role ใน context ให้ RequireRole ใช้)
func (h *Handler) RegisterRoutes(api gin.IRouter, authRequired gin.HandlerFunc) {
	api.PUT("/profile", authRequired, h.UpdateProfile)
	api.PUT("/profile/password", authRequired, h.ChangePassword)
	api.PUT("/profile/avatar", authRequired, h.UploadAvatar)
	api.DELETE("/profile/avatar", authRequired, h.DeleteAvatar)
	api.GET("/users/:id/avatar", authRequired, h.Avatar)

	ownerOnly := middleware.RequireRole(auth.RoleOwner)
	api.GET("/users", authRequired, ownerOnly, h.List)
	api.POST("/users", authRequired, ownerOnly, h.Create)
	api.PUT("/users/:id", authRequired, ownerOnly, h.Update)
	api.PUT("/users/:id/password", authRequired, ownerOnly, h.ResetPassword)
}

// ---------- โปรไฟล์ของตัวเอง ----------

// UpdateProfile PUT /api/v1/profile — แก้ชื่อของตัวเอง
func (h *Handler) UpdateProfile(c *gin.Context) {
	var req ProfileRequest
	if !bindValid(c, &req) {
		return
	}
	user, err := h.repo.UpdateProfile(c.Request.Context(), c.GetString(middleware.UserIDKey), req)
	writeUser(c, http.StatusOK, user, err)
}

// ChangePassword PUT /api/v1/profile/password — ต้องใส่รหัสเดิมถูกก่อน ได้ 204
func (h *Handler) ChangePassword(c *gin.Context) {
	var req PasswordRequest
	if !bindValid(c, &req) {
		return
	}
	userID := c.GetString(middleware.UserIDKey)
	hash, err := h.repo.PasswordHash(c.Request.Context(), userID)
	if err != nil {
		writeUser(c, http.StatusOK, User{}, err)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.CurrentPassword)) != nil {
		helpers.WriteError(c, http.StatusBadRequest, "password_current_invalid")
		return
	}
	h.savePassword(c, userID, req.NewPassword)
}

// UploadAvatar PUT /api/v1/profile/avatar — multipart/form-data ช่องชื่อ "avatar" (PNG / JPG / WebP ไม่เกิน 1 MB)
func (h *Handler) UploadAvatar(c *gin.Context) {
	data, contentType, problem := helpers.ReadImage(c, "avatar")
	if problem != "" {
		helpers.WriteError(c, http.StatusBadRequest, "avatar_"+problem)
		return
	}
	user, err := h.repo.SaveAvatar(c.Request.Context(), c.GetString(middleware.UserIDKey), Avatar{Data: data, ContentType: contentType})
	writeUser(c, http.StatusOK, user, err)
}

// DeleteAvatar DELETE /api/v1/profile/avatar — ได้ผู้ใช้ที่ avatar_version = 0
func (h *Handler) DeleteAvatar(c *gin.Context) {
	user, err := h.repo.DeleteAvatar(c.Request.Context(), c.GetString(middleware.UserIDKey))
	writeUser(c, http.StatusOK, user, err)
}

// Avatar GET /api/v1/users/:id/avatar — ทุกคนที่ login ดูรูปของกันได้ (cache นานเพราะ URL มี ?v=avatar_version)
func (h *Handler) Avatar(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "user_not_found")
	if !ok {
		return
	}
	avatar, err := h.repo.FindAvatar(c.Request.Context(), id)
	if errors.Is(err, ErrAvatarNotFound) {
		helpers.WriteError(c, http.StatusNotFound, "avatar_not_found")
		return
	}
	if err != nil {
		writeUser(c, http.StatusOK, User{}, err)
		return
	}
	c.Header("Cache-Control", "private, max-age=31536000, immutable")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, avatar.ContentType, avatar.Data)
}

// ---------- จัดการผู้ใช้ (เฉพาะเจ้าของร้าน) ----------

// List GET /api/v1/users
func (h *Handler) List(c *gin.Context) {
	users, err := h.repo.List(c.Request.Context())
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, users)
}

// Create POST /api/v1/users — เพิ่มผู้ใช้พร้อมรหัสผ่านแรก (บอกพนักงานให้ไปเปลี่ยนเองที่หน้าโปรไฟล์)
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if !bindValid(c, &req) {
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), h.hashCost)
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	user, err := h.repo.Create(c.Request.Context(), req, string(hash))
	writeUser(c, http.StatusCreated, user, err)
}

// Update PUT /api/v1/users/:id — แก้ชื่อ / role / เปิด-ปิดบัญชี (ร้านต้องเหลือเจ้าของที่ใช้งานได้อย่างน้อย 1 คน)
func (h *Handler) Update(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "user_not_found")
	if !ok {
		return
	}
	var req UpdateRequest
	if !bindValid(c, &req) {
		return
	}
	user, err := h.repo.Update(c.Request.Context(), id, req)
	writeUser(c, http.StatusOK, user, err)
}

// ResetPassword PUT /api/v1/users/:id/password — ตั้งรหัสใหม่ให้ผู้ใช้ (ไม่ต้องรู้รหัสเดิม) ได้ 204
func (h *Handler) ResetPassword(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "user_not_found")
	if !ok {
		return
	}
	var req ResetPasswordRequest
	if !bindValid(c, &req) {
		return
	}
	h.savePassword(c, id, req.NewPassword)
}

// savePassword hash รหัสใหม่แล้วบันทึก ได้ 204
func (h *Handler) savePassword(c *gin.Context, userID, password string) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.hashCost)
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	err = h.repo.UpdatePassword(c.Request.Context(), userID, string(hash))
	if err != nil {
		writeUser(c, http.StatusOK, User{}, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// validatable request ที่ตรวจตัวเองได้ (ทุก *Request ในไฟล์ model.go)
type validatable interface {
	Validate() string
}

// bindValid อ่าน JSON ใส่ req (ส่ง pointer มา) แล้วตรวจข้อมูล ไม่ผ่าน → ตอบ 400 แล้วคืน false
func bindValid(c *gin.Context, req validatable) bool {
	if c.ShouldBindJSON(req) != nil {
		helpers.WriteError(c, http.StatusBadRequest, "invalid_json")
		return false
	}
	if code := req.Validate(); code != "" {
		helpers.WriteError(c, http.StatusBadRequest, code)
		return false
	}
	return true
}

// writeUser ตอบผู้ใช้ / แปลง error ของ repository เป็น error code
func writeUser(c *gin.Context, status int, user User, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		helpers.WriteError(c, http.StatusNotFound, "user_not_found")
	case errors.Is(err, ErrEmailTaken):
		helpers.WriteError(c, http.StatusConflict, "user_email_taken")
	case errors.Is(err, ErrLastOwner):
		helpers.WriteError(c, http.StatusConflict, "user_last_owner")
	case err != nil:
		helpers.ServerError(c, err)
	default:
		c.JSON(status, user)
	}
}
