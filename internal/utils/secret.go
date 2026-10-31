package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

// ErrSecretInvalid ถอดรหัสไม่ได้ (ข้อมูลเสีย หรือเปลี่ยน JWT_SECRET ไปแล้ว)
var ErrSecretInvalid = errors.New("secret cannot be decrypted")

// Encrypt เข้ารหัสข้อความลับ (เช่น API key) ก่อนเก็บลง DB ด้วย AES-256-GCM
// กุญแจ = SHA-256 ของ secret (ใช้ JWT_SECRET) → ได้ข้อความ base64 (nonce + ข้อมูลที่เข้ารหัส)
func Encrypt(plain, secret string) (string, error) {
	gcm, err := newGCM(secret)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt ถอดรหัสข้อความจาก Encrypt (secret ต้องตัวเดียวกัน ไม่งั้นได้ ErrSecretInvalid)
func Decrypt(encoded, secret string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", ErrSecretInvalid
	}
	gcm, err := newGCM(secret)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", ErrSecretInvalid
	}
	nonce, sealed := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", ErrSecretInvalid
	}
	return string(plain), nil
}

func newGCM(secret string) (cipher.AEAD, error) {
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
