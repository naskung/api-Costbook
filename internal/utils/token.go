package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken token ไม่มี id ผู้ใช้อยู่ข้างใน
var ErrInvalidToken = errors.New("invalid token")

// CreateToken สร้าง token (JWT) หลัง login สำเร็จ ข้างในเก็บ id ผู้ใช้ และวันหมดอายุ
// token ถูกเซ็นด้วย secret → ถ้ามีคนแก้ข้อมูลใน token จะตรวจไม่ผ่าน
func CreateToken(userID, secret string, duration time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken ตรวจว่า token เซ็นด้วย secret ของเรา และยังไม่หมดอายุ แล้วคืน id ผู้ใช้
func ParseToken(tokenString, secret string) (string, error) {
	var claims jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return "", err
	}
	if claims.Subject == "" {
		return "", ErrInvalidToken
	}
	return claims.Subject, nil
}
