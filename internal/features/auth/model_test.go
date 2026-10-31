package auth

import "testing"

func TestLoginRequest_Validate(t *testing.T) {
	tests := []struct {
		name string
		req  LoginRequest
		want string
	}{
		{"กรอกครบ ผ่าน", LoginRequest{Email: "admin@costbook.com", Password: "1234"}, ""},
		{"อีเมลว่าง ไม่ผ่าน", LoginRequest{Email: "", Password: "1234"}, "email_required"},
		{"อีเมลมีแต่ช่องว่าง ไม่ผ่าน", LoginRequest{Email: "   ", Password: "1234"}, "email_required"},
		{"รหัสผ่านว่าง ไม่ผ่าน", LoginRequest{Email: "admin@costbook.com", Password: ""}, "password_required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.req.Validate()
			if got != tt.want {
				t.Errorf("Validate() = %q, ต้องได้ %q", got, tt.want)
			}
		})
	}
}
