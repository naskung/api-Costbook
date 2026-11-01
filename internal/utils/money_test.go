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

func TestNetProfit(t *testing.T) {
	tests := []struct {
		name        string
		revenue     float64
		costOfSales float64
		expense     float64
		want        float64
	}{
		{"ขาย 1,000 ต้นทุน 600 รายจ่าย 100 กำไร 300", 1000, 600, 100, 300},
		{"ขายได้แต่ยังไม่หักต้นทุน ไม่ใช่กำไรเต็มยอดขาย", 239.13, 227.74, 0, 11.39},
		{"ต้นทุน + รายจ่ายมากกว่ารายได้ ขาดทุนเป็นค่าติดลบ", 100, 80, 70.5, -50.5},
		{"0.1 + 0.2 ลบ 0.3 ได้ 0 พอดี (ปัดเศษแล้ว)", 0.1 + 0.2, 0.3, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NetProfit(tt.revenue, tt.costOfSales, tt.expense)
			if got != tt.want {
				t.Errorf("NetProfit(%v, %v, %v) = %v, ต้องได้ %v", tt.revenue, tt.costOfSales, tt.expense, got, tt.want)
			}
		})
	}
}
