package transaction

import "github.com/naskung/api-Costbook/internal/utils"

// NewSummary คิดรายได้ / กำไรขั้นต้น / รายจ่ายที่หักในกำไร / กำไรสุทธิ จากยอดของช่วงวันที่
// หน้าภาพรวมใช้ตัวนี้ด้วย → 2 หน้าคิดกำไรแบบเดียวกันเสมอ
func NewSummary(period utils.DateRange, a Amounts) Summary {
	revenue := utils.Round2(a.Sales + a.OtherIncome)
	operatingExpense := utils.Round2(a.Expense - a.MaterialExpense)
	return Summary{
		From:             period.From,
		To:               period.To,
		Income:           utils.Round2(a.Income),
		Expense:          utils.Round2(a.Expense),
		Sales:            utils.Round2(a.Sales),
		OtherIncome:      utils.Round2(a.OtherIncome),
		Revenue:          revenue,
		CostOfSales:      utils.Round2(a.CostOfSales),
		GrossProfit:      utils.Round2(a.Sales - a.CostOfSales),
		MaterialExpense:  utils.Round2(a.MaterialExpense),
		OperatingExpense: operatingExpense,
		Profit:           utils.NetProfit(revenue, a.CostOfSales, operatingExpense),
	}
}
