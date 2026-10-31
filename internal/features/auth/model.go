package auth

import "strings"

// User ข้อมูลผู้ใช้ (ชื่อ json ต้องตรงกับ model ฝั่ง Flutter)
// PasswordHash มี json:"-" → ไม่ถูกส่งออกไปใน JSON เด็ดขาด
type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	PasswordHash string `json:"-"`
}

// LoginRequest ข้อมูลที่รับมาตอน login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse ข้อมูลที่ตอบกลับเมื่อ login สำเร็จ
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// Validate เช็คว่ากรอกครบไหม คืน error code ("" = ผ่าน)
func (req LoginRequest) Validate() string {
	if strings.TrimSpace(req.Email) == "" {
		return "email_required"
	}
	if req.Password == "" {
		return "password_required"
	}
	return ""
}
