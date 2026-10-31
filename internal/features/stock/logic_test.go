package stock

import "testing"

func TestAverageCost(t *testing.T) {
	tests := []struct {
		name     string
		oldStock int
		oldCost  float64
		quantity int
		unitCost float64
		want     float64
	}{
		{"เดิม 10 ทุน 5 + รับ 10 ทุน 6 ได้ทุนเฉลี่ย 5.50", 10, 5, 10, 6, 5.5},
		{"เดิม 3 ทุน 10 + รับ 1 ทุน 20 ได้ทุนเฉลี่ย 12.50", 3, 10, 1, 20, 12.5},
		{"ของเดิมหมด ใช้ทุนใหม่เลย", 0, 99, 5, 7.25, 7.25},
		{"หารไม่ลงตัว ปัด 2 ตำแหน่ง: เดิม 2 ทุน 1 + รับ 1 ทุน 2 = 1.33", 2, 1, 1, 2, 1.33},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := averageCost(tt.oldStock, tt.oldCost, tt.quantity, tt.unitCost)
			if got != tt.want {
				t.Errorf("averageCost(%d, %v, %d, %v) = %v, ต้องได้ %v", tt.oldStock, tt.oldCost, tt.quantity, tt.unitCost, got, tt.want)
			}
		})
	}
}
