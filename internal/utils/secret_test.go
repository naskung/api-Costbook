package utils

import (
	"errors"
	"strings"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	encrypted, err := Encrypt("sk-ant-api03-abcdef", "secret-1")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(encrypted, "sk-ant") {
		t.Error("ข้อความที่เข้ารหัสแล้วต้องไม่เห็น key ตัวจริง")
	}
	plain, err := Decrypt(encrypted, "secret-1")
	if err != nil || plain != "sk-ant-api03-abcdef" {
		t.Errorf("ถอดรหัสได้ %q %v ต้องได้ key เดิม", plain, err)
	}
}

func TestEncrypt_DifferentEachTime(t *testing.T) {
	first, _ := Encrypt("same", "secret-1")
	second, _ := Encrypt("same", "secret-1")
	if first == second {
		t.Error("เข้ารหัสข้อความเดิม 2 ครั้ง ต้องได้ผลไม่เหมือนกัน (สุ่ม nonce)")
	}
}

func TestDecrypt_Invalid(t *testing.T) {
	encrypted, _ := Encrypt("sk-ant-api03-abcdef", "secret-1")
	tests := map[string]string{
		"secret ไม่ตรง (เปลี่ยน JWT_SECRET)": encrypted,
		"ไม่ใช่ base64":                      "@@@",
		"สั้นเกิน":                           "YWJj",
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Decrypt(input, "secret-2"); !errors.Is(err, ErrSecretInvalid) {
				t.Errorf("ต้องได้ ErrSecretInvalid ได้ %v", err)
			}
		})
	}
}
