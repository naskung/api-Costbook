package utils

import (
	"testing"
	"time"
)

func TestCreateToken_ParseToken(t *testing.T) {
	token, err := CreateToken("user-1", "secret", time.Hour)
	if err != nil {
		t.Fatalf("CreateToken() error = %v, ต้องไม่มี error", err)
	}

	userID, err := ParseToken(token, "secret")
	if err != nil {
		t.Fatalf("ParseToken() error = %v, ต้องไม่มี error", err)
	}
	if userID != "user-1" {
		t.Errorf("ParseToken() = %q, ต้องได้ %q", userID, "user-1")
	}
}

func TestParseToken_Invalid(t *testing.T) {
	expiredToken, _ := CreateToken("user-1", "secret", -time.Minute)
	otherSecretToken, _ := CreateToken("user-1", "other-secret", time.Hour)
	noUserToken, _ := CreateToken("", "secret", time.Hour)

	tests := []struct {
		name  string
		token string
	}{
		{"token หมดอายุ ได้ error", expiredToken},
		{"token เซ็นด้วย secret อื่น ได้ error", otherSecretToken},
		{"token ไม่มี id ผู้ใช้ ได้ error", noUserToken},
		{"ข้อความมั่ว ได้ error", "not-a-token"},
		{"ค่าว่าง ได้ error", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseToken(tt.token, "secret")
			if err == nil {
				t.Error("ParseToken() error = nil, ต้องได้ error")
			}
		})
	}
}
