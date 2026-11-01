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

func TestMaterialNeed(t *testing.T) {
	tests := []struct {
		name          string
		perBatch      float64
		batchQuantity int
		quantity      int
		want          int
	}{
		{"เคส 1 ชิ้น/รอบ ใช้ TPU 30 กรัม ผลิต 10 ชิ้น = 300 กรัม", 30, 1, 10, 300},
		{"พวงกุญแจ 6 ชิ้น/รอบ ใช้ PLA 60 กรัม ผลิต 4 ชิ้น = 40 กรัม", 60, 6, 4, 40},
		{"ห่วง 6 อัน/รอบ ผลิต 1 ชิ้น = 1 อัน", 6, 6, 1, 1},
		{"เศษปัดตอนรวม 2.5 กรัม × 3 = 7.5 → 8", 2.5, 1, 3, 8},
		{"จำนวนต่อรอบเป็น 0 (ข้อมูลผิด) คิดเป็น 1 ชิ้น/รอบ", 10, 0, 2, 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := materialNeed(tt.perBatch, tt.batchQuantity, tt.quantity); got != tt.want {
				t.Errorf("materialNeed() = %d, ต้องได้ %d", got, tt.want)
			}
		})
	}
}

func TestWithSaleProfit(t *testing.T) {
	tests := []struct {
		name         string
		quantity     int
		lineQuantity int
		lineNet      float64
		lineCost     float64
		want         Movement
	}{
		{"ขาย 2 ชิ้น ยอดก่อน VAT 250.52 ทุนรวม 227.74 → กำไร 22.78", 2, 2, 250.52, 227.74,
			Movement{Quantity: 2, HasProfit: true, SaleAmount: 250.52, CostAmount: 227.74, Profit: 22.78}},
		{"บิลมีสินค้าเดียวกัน 2 บรรทัด (รวม 3 ชิ้น) แถวนี้ 1 ชิ้น ได้ส่วน 1 ใน 3", 1, 3, 300, 150,
			Movement{Quantity: 1, HasProfit: true, SaleAmount: 100, CostAmount: 50, Profit: 50}},
		{"ขายต่ำกว่าทุน กำไรติดลบ", 1, 1, 80, 100,
			Movement{Quantity: 1, HasProfit: true, SaleAmount: 80, CostAmount: 100, Profit: -20}},
		{"ไม่ใช่แถวที่ขาย (lineQuantity = 0) ไม่มีกำไร", 5, 0, 0, 0, Movement{Quantity: 5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := withSaleProfit(Movement{Quantity: tt.quantity}, tt.lineQuantity, tt.lineNet, tt.lineCost)
			if got != tt.want {
				t.Errorf("ได้ %+v ต้องได้ %+v", got, tt.want)
			}
		})
	}
}
