package utils

import "testing"

func TestRound2(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  float64
	}{
		{"10.456 ปัดขึ้นเป็น 10.46", 10.456, 10.46},
		{"10.454 ปัดลงเป็น 10.45", 10.454, 10.45},
		{"10 ได้ 10 เท่าเดิม", 10, 10},
		{"0 ได้ 0", 0, 0},
		{"ติดลบ -2.345 ได้ -2.35", -2.345, -2.35},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Round2(tt.value)
			if got != tt.want {
				t.Errorf("Round2(%v) = %v, ต้องได้ %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestProfit(t *testing.T) {
	tests := []struct {
		name    string
		income  float64
		expense float64
		want    float64
	}{
		{"รายรับ 1,000 รายจ่าย 400 กำไร 600", 1000, 400, 600},
		{"รายจ่ายมากกว่ารายรับ ขาดทุนเป็นค่าติดลบ", 100, 250.5, -150.5},
		{"0.1 + 0.2 ลบ 0.3 ได้ 0 พอดี (ปัดเศษแล้ว)", 0.1 + 0.2, 0.3, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Profit(tt.income, tt.expense)
			if got != tt.want {
				t.Errorf("Profit(%v, %v) = %v, ต้องได้ %v", tt.income, tt.expense, got, tt.want)
			}
		})
	}
}
