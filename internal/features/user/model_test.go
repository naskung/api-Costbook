package user

import (
	"strings"
	"testing"

	"github.com/naskung/api-Costbook/internal/features/auth"
)

func TestCreateRequest_Validate(t *testing.T) {
	valid := CreateRequest{Email: " Staff@CostBook.com ", FirstName: "สมชาย", Role: auth.RoleStaff, Password: "123456"}
	tests := []struct {
		name string
		edit func(r *CreateRequest)
		want string
	}{
		{"ครบ ผ่าน (อีเมลตัวใหญ่ / มีช่องว่างได้ เก็บเป็นตัวเล็ก)", func(r *CreateRequest) {}, ""},
		{"อีเมลผิดรูปแบบ", func(r *CreateRequest) { r.Email = "staff" }, "user_email_invalid"},
		{"ไม่มีชื่อ", func(r *CreateRequest) { r.FirstName = "  " }, "user_name_required"},
		{"ชื่อยาวเกิน", func(r *CreateRequest) { r.FirstName = strings.Repeat("ก", maxNameLength+1) }, "user_name_too_long"},
		{"role ไม่มีจริง", func(r *CreateRequest) { r.Role = "admin" }, "user_role_invalid"},
		{"รหัสสั้นเกิน 6 ตัว", func(r *CreateRequest) { r.Password = "12345" }, "password_too_short"},
		{"รหัสภาษาไทย 6 ตัว นับเป็นตัวอักษรไม่ใช่ byte", func(r *CreateRequest) { r.Password = "รหัสผ่าน" }, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := valid
			tt.edit(&req)
			if got := req.Validate(); got != tt.want {
				t.Errorf("ได้ %q ต้องได้ %q", got, tt.want)
			}
		})
	}
}

func TestOtherRequests_Validate(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"โปรไฟล์ ไม่มีชื่อ", ProfileRequest{}.Validate(), "user_name_required"},
		{"โปรไฟล์ ครบ", ProfileRequest{FirstName: "สมชาย"}.Validate(), ""},
		{"เปลี่ยนรหัส ไม่ใส่รหัสเดิม", PasswordRequest{NewPassword: "123456"}.Validate(), "password_current_required"},
		{"เปลี่ยนรหัส รหัสใหม่สั้น", PasswordRequest{CurrentPassword: "1234", NewPassword: "123"}.Validate(), "password_too_short"},
		{"แก้ผู้ใช้ role ผิด", UpdateRequest{FirstName: "ก", Role: "boss"}.Validate(), "user_role_invalid"},
		{"แก้ผู้ใช้ ครบ", UpdateRequest{FirstName: "ก", Role: auth.RoleManager}.Validate(), ""},
		{"ตั้งรหัสให้ผู้ใช้ สั้นเกิน", ResetPasswordRequest{NewPassword: "1"}.Validate(), "password_too_short"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: ได้ %q ต้องได้ %q", tt.name, tt.got, tt.want)
		}
	}
}

func TestStopsBeingOwner(t *testing.T) {
	owner := User{Role: auth.RoleOwner, Active: true}
	tests := []struct {
		name    string
		current User
		role    string
		active  bool
		want    bool
	}{
		{"เจ้าของ → ผู้จัดการ", owner, auth.RoleManager, true, true},
		{"เจ้าของ ถูกปิดบัญชี", owner, auth.RoleOwner, false, true},
		{"เจ้าของ แก้แค่ชื่อ", owner, auth.RoleOwner, true, false},
		{"เจ้าของที่ถูกปิดอยู่แล้ว เปลี่ยนเป็นพนักงาน", User{Role: auth.RoleOwner}, auth.RoleStaff, false, false},
		{"พนักงาน → ผู้จัดการ", User{Role: auth.RoleStaff, Active: true}, auth.RoleManager, true, false},
	}
	for _, tt := range tests {
		if got := stopsBeingOwner(tt.current, tt.role, tt.active); got != tt.want {
			t.Errorf("%s: ได้ %v ต้องได้ %v", tt.name, got, tt.want)
		}
	}
}
