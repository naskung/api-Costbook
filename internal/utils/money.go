package utils

import "math"

// Round2 ปัดเงินให้เหลือทศนิยม 2 ตำแหน่ง เช่น 10.456 → 10.46
// ใช้ทุกครั้งที่คำนวณเงินเสร็จ ก่อนบันทึกหรือส่งให้ Flutter
func Round2(value float64) float64 {
	return math.Round(value*100) / 100
}

// Profit กำไร = รายรับ − รายจ่าย (ใช้ทั้งสรุปรายเดือนและหน้าภาพรวม)
func Profit(income, expense float64) float64 {
	return Round2(income - expense)
}
