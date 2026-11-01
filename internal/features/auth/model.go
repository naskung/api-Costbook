package auth

import "strings"

// role ของผู้ใช้ (ตรงกับ CHECK ในตาราง users — migration 000014)
// owner = เจ้าของร้าน ทำได้ทุกอย่าง + จัดการผู้ใช้, manager = ผู้จัดการ, staff = พนักงาน
const (
	RoleOwner   = "owner"
	RoleManager = "manager"
	RoleStaff   = "staff"
)

// Roles role ทั้งหมด เรียงจากสิทธิ์มากไปน้อย
var Roles = []string{RoleOwner, RoleManager, RoleStaff}

// User ข้อมูลผู้ใช้ (ชื่อ json ต้องตรงกับ model ฝั่ง Flutter)
// PasswordHash / Active มี json:"-" → ไม่ถูกส่งออกไปใน JSON (ผู้ใช้ที่ login ได้ active เสมอ)
// AvatarVersion = เวลาที่เปลี่ยนรูปโปรไฟล์ล่าสุด (0 = ไม่มีรูป) — รูปอยู่ที่ /users/{id}/avatar?v=<avatar_version>
type User struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	Role          string `json:"role"`
	AvatarVersion int64  `json:"avatar_version"`
	Active        bool   `json:"-"`
	PasswordHash  string `json:"-"`
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
