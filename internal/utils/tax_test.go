package utils

import "testing"

func TestVAT(t *testing.T) {
	tests := []struct {
		name  string
		price float64
		want  float64
	}{
		{"100 บาท ได้ VAT 7 บาท", 100, 7},
		{"0 บาท ได้ VAT 0 บาท", 0, 0},
		{"1,234.56 บาท ได้ VAT 86.42 บาท", 1234.56, 86.42},
		{"0.10 บาท ได้ VAT 0.01 บาท", 0.10, 0.01},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VAT(tt.price)
			if got != tt.want {
				t.Errorf("VAT(%v) = %v, ต้องได้ %v", tt.price, got, tt.want)
			}
		})
	}
}
