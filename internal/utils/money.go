package utils

import "math"

// Round2 ปัดเงินให้เหลือทศนิยม 2 ตำแหน่ง เช่น 10.456 → 10.46
// ใช้ทุกครั้งที่คำนวณเงินเสร็จ ก่อนบันทึกหรือส่งให้ Flutter
func Round2(value float64) float64 {
	return math.Round(value*100) / 100
}

// NetProfit กำไรสุทธิ = รายได้ − ต้นทุนสินค้าที่ขาย − รายจ่าย (ไม่รวมซื้อวัตถุดิบ เพราะคิดอยู่ในต้นทุนสินค้าแล้ว)
// ใช้ทั้งสรุปรายรับ-รายจ่ายและหน้าภาพรวม
func NetProfit(revenue, costOfSales, operatingExpense float64) float64 {
	return Round2(revenue - costOfSales - operatingExpense)
}
