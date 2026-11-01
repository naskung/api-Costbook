package utils

import "strings"

// IsTaxID เลขผู้เสียภาษีต้องเป็นตัวเลข 13 หลัก ไม่มีขีด
func IsTaxID(value string) bool {
	if len(value) != 13 {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

// IsUUID เช็คว่าเป็น id แบบ UUID เช่น "a00d265d-5466-4cba-aba7-005953ae30fc" ไหม
// ใช้เช็ค id จาก URL ก่อนส่งให้ DB (ถ้าไม่เช็ค Postgres จะ error แล้วกลายเป็น 500)
func IsUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, char := range value {
		isDashPosition := i == 8 || i == 13 || i == 18 || i == 23
		if isDashPosition {
			if char != '-' {
				return false
			}
			continue
		}
		if !strings.ContainsRune("0123456789abcdefABCDEF", char) {
			return false
		}
	}
	return true
}

// IsEmail อีเมลรูปแบบ ชื่อ@โดเมน.นามสกุล (ไม่มีช่องว่าง มี @ ตัวเดียว)
func IsEmail(value string) bool {
	name, domain, found := strings.Cut(value, "@")
	if !found || name == "" || strings.ContainsAny(value, " \t\n") || strings.Contains(domain, "@") {
		return false
	}
	dot := strings.LastIndex(domain, ".")
	return dot > 0 && dot < len(domain)-1
}
