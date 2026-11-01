package user

import (
	"slices"
	"strings"
	"time"

	"github.com/naskung/api-Costbook/internal/features/auth"
	"github.com/naskung/api-Costbook/internal/utils"
)

// minPasswordLength รหัสผ่านใหม่ต้องยาวอย่างน้อยเท่านี้
const minPasswordLength = 6

// maxNameLength ชื่อ / นามสกุล ยาวไม่เกินเท่านี้ (ตัวอักษร)
const maxNameLength = 100

// User ผู้ใช้ในหน้าจัดการผู้ใช้ (เจ้าของร้านดู) — ไม่มี password_hash ออกไปเด็ดขาด
type User struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Role          string    `json:"role"`
	Active        bool      `json:"active"`
	AvatarVersion int64     `json:"avatar_version"`
	CreatedAt     time.Time `json:"created_at"`
}

// Avatar รูปโปรไฟล์ (เก็บใน DB แบบเดียวกับรูปสินค้า)
type Avatar struct {
	Data        []byte
	ContentType string
}

// ProfileRequest แก้ชื่อของตัวเอง (PUT /profile)
type ProfileRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// PasswordRequest เปลี่ยนรหัสผ่านของตัวเอง (PUT /profile/password) ต้องใส่รหัสเดิมด้วย
type PasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// CreateRequest เจ้าของร้านเพิ่มผู้ใช้ (POST /users) พร้อมตั้งรหัสผ่านแรกให้
type CreateRequest struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
	Password  string `json:"password"`
}

// UpdateRequest เจ้าของร้านแก้ผู้ใช้ (PUT /users/{id}) — ไม่ส่ง active = ใช้ค่าเดิม
type UpdateRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
	Active    *bool  `json:"active"`
}

// ResetPasswordRequest เจ้าของร้านตั้งรหัสผ่านใหม่ให้ผู้ใช้ (PUT /users/{id}/password) เช่น พนักงานลืมรหัส
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

// Validate ชื่อต้องมี ไม่ยาวเกิน คืน error code ("" = ผ่าน)
func (r ProfileRequest) Validate() string {
	return validateName(r.FirstName, r.LastName)
}

func (r PasswordRequest) Validate() string {
	if r.CurrentPassword == "" {
		return "password_current_required"
	}
	return validatePassword(r.NewPassword)
}

func (r CreateRequest) Validate() string {
	if !utils.IsEmail(normalizeEmail(r.Email)) {
		return "user_email_invalid"
	}
	if code := validateName(r.FirstName, r.LastName); code != "" {
		return code
	}
	if !slices.Contains(auth.Roles, r.Role) {
		return "user_role_invalid"
	}
	return validatePassword(r.Password)
}

func (r UpdateRequest) Validate() string {
	if code := validateName(r.FirstName, r.LastName); code != "" {
		return code
	}
	if !slices.Contains(auth.Roles, r.Role) {
		return "user_role_invalid"
	}
	return ""
}

func (r ResetPasswordRequest) Validate() string {
	return validatePassword(r.NewPassword)
}

func validateName(firstName, lastName string) string {
	if strings.TrimSpace(firstName) == "" {
		return "user_name_required"
	}
	if len([]rune(firstName)) > maxNameLength || len([]rune(lastName)) > maxNameLength {
		return "user_name_too_long"
	}
	return ""
}

func validatePassword(password string) string {
	if len([]rune(password)) < minPasswordLength {
		return "password_too_short"
	}
	return ""
}

// normalizeEmail อีเมลเก็บเป็นตัวเล็กไม่มีช่องว่างหน้าหลัง (ตอน login ก็แปลงแบบเดียวกัน)
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
