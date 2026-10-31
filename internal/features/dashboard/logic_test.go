package dashboard

import (
	"reflect"
	"testing"
)

func TestPercentOf(t *testing.T) {
	tests := []struct {
		name        string
		part, total float64
		want        float64
	}{
		{"กำไร 300 จากยอด 1,200 = 25%", 300, 1200, 25},
		{"ยอดรวม 0 ได้ 0 ไม่หารด้วย 0", 50, 0, 0},
		{"ขาดทุน -100 จาก 400 = -25%", -100, 400, -25},
		{"ปัดเศษ 2 ตำแหน่ง 1/3 = 33.33", 1, 3, 33.33},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := percentOf(tt.part, tt.total); got != tt.want {
				t.Errorf("ได้ %v ต้องได้ %v", got, tt.want)
			}
		})
	}
}

func TestSalesSummary(t *testing.T) {
	got := salesSummary(SalesSummary{NetSales: 1355.14, Cost: 452.8}, 2)
	if got.GrossProfit != 902.34 || got.GrossMargin != 66.59 || got.AverageBill != 677.57 {
		t.Errorf("ได้ %+v ต้องได้ กำไร 902.34 margin 66.59 เฉลี่ยต่อบิล 677.57", got)
	}

	empty := salesSummary(SalesSummary{}, 0)
	if empty.GrossMargin != 0 || empty.AverageBill != 0 {
		t.Errorf("ไม่มีบิล ได้ %+v ต้องได้ 0 ทุกช่อง", empty)
	}
}

func TestProductSales(t *testing.T) {
	got := productSales(ProductSales{NetSales: 934.58, Cost: 211.2})
	if got.Profit != 723.38 || got.Margin != 77.4 {
		t.Errorf("ได้ %+v ต้องได้ กำไร 723.38 margin 77.4", got)
	}
}

func TestWithPercent(t *testing.T) {
	got := withPercent([]CategoryAmount{{Category: "rent", Amount: 5000}, {Category: "utilities", Amount: 3000}, {Category: "", Amount: 2000}})
	want := []CategoryAmount{{"rent", 5000, 50}, {"utilities", 3000, 30}, {"", 2000, 20}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ได้ %+v ต้องได้ %+v", got, want)
	}
	if len(withPercent([]CategoryAmount{})) != 0 {
		t.Error("ไม่มีหมวด ต้องได้ list ว่าง")
	}
}

func TestCustomerPrice(t *testing.T) {
	tests := []struct {
		name                          string
		vatRegistered, pricesIncludeV bool
		want                          float64
	}{
		{"จด VAT ราคาไม่รวม VAT: 100 → ลูกค้าจ่าย 107", true, false, 107},
		{"จด VAT ราคารวม VAT แล้ว: ลูกค้าจ่าย 100", true, true, 100},
		{"ไม่จด VAT: ลูกค้าจ่าย 100", false, false, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := customerPrice(100, tt.vatRegistered, tt.pricesIncludeV); got != tt.want {
				t.Errorf("ได้ %v ต้องได้ %v", got, tt.want)
			}
		})
	}
}

func TestMarketPosition(t *testing.T) {
	tests := []struct {
		name             string
		price, low, high float64
		want             string
	}{
		{"ถูกกว่าราคาต่ำสุด = below", 150, 199, 390, positionBelow},
		{"อยู่ในช่วง = within", 250, 199, 390, positionWithin},
		{"เท่าราคาต่ำสุดพอดี = within", 199, 199, 390, positionWithin},
		{"แพงกว่าราคาสูงสุด = above", 450, 199, 390, positionAbove},
		{"ไม่มีราคาตลาด = unknown", 250, 0, 0, positionUnknown},
		{"รู้แค่ราคาต่ำสุด แพงกว่า = within (ไม่รู้ขอบบน)", 999, 199, 0, positionWithin},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := marketPosition(tt.price, tt.low, tt.high); got != tt.want {
				t.Errorf("ได้ %q ต้องได้ %q", got, tt.want)
			}
		})
	}
}

func TestMarketPositionOf(t *testing.T) {
	m := marketPositionOf(MarketPosition{RetailLow: 199, RetailHigh: 390, RetailTypical: 250}, 250, true, true)
	if m.CustomerPrice != 250 || m.Position != positionWithin || m.DiffPercent != 0 {
		t.Errorf("ราคาเท่าราคาที่เจอบ่อย ได้ %+v", m)
	}

	m = marketPositionOf(MarketPosition{RetailLow: 199, RetailHigh: 390, RetailTypical: 250}, 280, true, false)
	if m.CustomerPrice != 299.6 || m.DiffPercent != 19.84 {
		t.Errorf("ราคา 280 + VAT = 299.60 แพงกว่า 250 อยู่ 19.84%% ได้ %+v", m)
	}
}

func TestPreviousMonth(t *testing.T) {
	trend := []MonthTotals{{Month: "2026-09", Income: 500}, {Month: "2026-10", Income: 900}}
	if got := previousMonth(trend); got.Month != "2026-09" {
		t.Errorf("ได้ %+v ต้องได้เดือน 2026-09", got)
	}
	if got := previousMonth(trend[:1]); got.Month != "" {
		t.Errorf("มีเดือนเดียว ได้ %+v ต้องได้ค่าว่าง", got)
	}
}

func TestPreviousMonthStart(t *testing.T) {
	tests := map[string]string{
		"2026-10-01": "2026-09-01",
		"2026-01-01": "2025-12-01",
		"2026-03-01": "2026-02-01",
		"abc":        "abc",
	}
	for input, want := range tests {
		if got := previousMonthStart(input); got != want {
			t.Errorf("previousMonthStart(%q) = %q ต้องได้ %q", input, got, want)
		}
	}
}
