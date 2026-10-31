package utils

import "testing"

func TestCostPerUnit(t *testing.T) {
	tests := []struct {
		name     string
		material float64
		labor    float64
		overhead float64
		quantity int
		want     float64
	}{
		{"ต้นทุนรวม 1,000 บาท ผลิต 100 ชิ้น ได้ชิ้นละ 10 บาท", 500, 300, 200, 100, 10},
		{"ต้นทุนรวม 100 บาท ผลิต 3 ชิ้น ได้ชิ้นละ 33.33 บาท", 100, 0, 0, 3, 33.33},
		{"จำนวนผลิต 0 ได้ 0 (ไม่หารด้วย 0)", 500, 0, 0, 0, 0},
		{"จำนวนผลิตติดลบ ได้ 0", 500, 0, 0, -5, 0},
		{"ไม่มีต้นทุนเลย ได้ 0", 0, 0, 0, 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CostPerUnit(tt.material, tt.labor, tt.overhead, tt.quantity)
			if got != tt.want {
				t.Errorf("CostPerUnit(%v, %v, %v, %v) = %v, ต้องได้ %v", tt.material, tt.labor, tt.overhead, tt.quantity, got, tt.want)
			}
		})
	}
}
