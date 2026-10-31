package utils

import "time"

// DateLayout รูปแบบวันที่ที่รับ / ส่งกับ Flutter เช่น "2026-10-31"
const DateLayout = "2006-01-02"

// IsDate เช็คว่าเป็นวันที่ที่มีอยู่จริงในรูปแบบ "2026-10-31" ไหม
func IsDate(value string) bool {
	_, err := time.Parse(DateLayout, value)
	return err == nil
}
