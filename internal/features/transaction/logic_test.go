package transaction

import (
	"testing"

	"github.com/naskung/api-Costbook/internal/utils"
)

func TestNewSummary(t *testing.T) {
	tests := []struct {
		name    string
		amounts Amounts
		want    Summary
	}{
		{
			"ขายอย่างเดียว กำไร = ยอดขาย − ต้นทุน ไม่ใช่ยอดขายเต็ม",
			Amounts{Income: 239.13, Sales: 239.13, CostOfSales: 227.74},
			Summary{From: "2026-11-01", To: "2026-11-30", Income: 239.13, Sales: 239.13, Revenue: 239.13, CostOfSales: 227.74, GrossProfit: 11.39, Profit: 11.39},
		},
		{
			"มีรายรับอื่น + รายจ่าย: รายได้รวมรายรับอื่น และหักรายจ่าย",
			Amounts{Income: 1300, Expense: 400, Sales: 1000, OtherIncome: 300, CostOfSales: 600},
			Summary{From: "2026-11-01", To: "2026-11-30", Income: 1300, Expense: 400, Sales: 1000, OtherIncome: 300, Revenue: 1300, CostOfSales: 600, GrossProfit: 400, OperatingExpense: 400, Profit: 300},
		},
		{
			"ซื้อวัตถุดิบไม่หักในกำไร (คิดในต้นทุนสินค้าแล้ว) แต่ยังนับเป็นเงินออก",
			Amounts{Income: 1000, Expense: 1500, Sales: 1000, CostOfSales: 600, MaterialExpense: 1250},
			Summary{From: "2026-11-01", To: "2026-11-30", Income: 1000, Expense: 1500, Sales: 1000, Revenue: 1000, CostOfSales: 600, GrossProfit: 400, MaterialExpense: 1250, OperatingExpense: 250, Profit: 150},
		},
		{
			"รายจ่ายมากกว่ากำไรขั้นต้น ขาดทุนเป็นค่าติดลบ",
			Amounts{Expense: 5000, Sales: 1000, CostOfSales: 600},
			Summary{From: "2026-11-01", To: "2026-11-30", Expense: 5000, Sales: 1000, Revenue: 1000, CostOfSales: 600, GrossProfit: 400, OperatingExpense: 5000, Profit: -4600},
		},
		{"เดือนว่าง ได้ 0 ทั้งหมด", Amounts{}, Summary{From: "2026-11-01", To: "2026-11-30"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSummary(utils.DateRange{From: "2026-11-01", To: "2026-11-30"}, tt.amounts)
			if got != tt.want {
				t.Errorf("ได้ %+v\nต้องได้ %+v", got, tt.want)
			}
		})
	}
}
