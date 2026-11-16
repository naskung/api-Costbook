package dashboard

import (
	"time"

	"github.com/naskung/api-Costbook/internal/features/transaction"
	"github.com/naskung/api-Costbook/internal/utils"
)

// ตำแหน่งราคาของร้านเทียบกับตลาด
const (
	positionBelow   = "below"
	positionWithin  = "within"
	positionAbove   = "above"
	positionUnknown = "unknown"
)

// percentOf part คิดเป็นกี่ % ของ total (total = 0 → 0)
// เช่น กำไร 300 จากยอดขาย 1,200 → 25
func percentOf(part, total float64) float64 {
	if total == 0 {
		return 0
	}
	return utils.Round2(part / total * 100)
}

// salesSummary คิดกำไรขั้นต้น / margin / ยอดเฉลี่ยต่อบิล จากยอดขายและต้นทุน
func salesSummary(s SalesSummary, bills int) SalesSummary {
	s.NetSales = utils.Round2(s.NetSales)
	s.Cost = utils.Round2(s.Cost)
	s.GrossProfit = utils.Round2(s.NetSales - s.Cost)
	s.GrossMargin = percentOf(s.GrossProfit, s.NetSales)
	if bills > 0 {
		s.AverageBill = utils.Round2(s.NetSales / float64(bills))
	}
	return s
}

// productSales คิดกำไร + margin ของสินค้าขายดี
func productSales(p ProductSales) ProductSales {
	p.NetSales = utils.Round2(p.NetSales)
	p.Cost = utils.Round2(p.Cost)
	p.Profit = utils.Round2(p.NetSales - p.Cost)
	p.Margin = percentOf(p.Profit, p.NetSales)
	return p
}

// withPercent ใส่ % ของแต่ละหมวด เทียบกับผลรวมทุกหมวด
func withPercent(items []CategoryAmount) []CategoryAmount {
	total := 0.0
	for _, item := range items {
		total += item.Amount
	}
	for i := range items {
		items[i].Amount = utils.Round2(items[i].Amount)
		items[i].Percent = percentOf(items[i].Amount, total)
	}
	return items
}

// customerPrice ราคาที่ลูกค้าจ่ายจริง (ราคาตลาดที่เห็นตามเว็บคือราคาที่ลูกค้าจ่าย)
// ร้านจด VAT และตั้งราคาไม่รวม VAT → บวก VAT 7% ก่อนเอาไปเทียบ
func customerPrice(sellPrice float64, vatRegistered, pricesIncludeVAT bool) float64 {
	if vatRegistered && !pricesIncludeVAT {
		return utils.Round2(sellPrice + utils.VAT(sellPrice))
	}
	return utils.Round2(sellPrice)
}

// marketPosition ราคาของร้านอยู่ตรงไหนของช่วงราคาตลาด (ไม่มีช่วงราคา → unknown)
func marketPosition(price, low, high float64) string {
	if low <= 0 && high <= 0 {
		return positionUnknown
	}
	if price < low {
		return positionBelow
	}
	if high > 0 && price > high {
		return positionAbove
	}
	return positionWithin
}

// marketPositionOf เติมราคาลูกค้า / ตำแหน่ง / % ต่างจากราคาที่เจอบ่อย
func marketPositionOf(m MarketPosition, sellPrice float64, vatRegistered, pricesIncludeVAT bool) MarketPosition {
	m.CustomerPrice = customerPrice(sellPrice, vatRegistered, pricesIncludeVAT)
	m.Position = marketPosition(m.CustomerPrice, m.RetailLow, m.RetailHigh)
	if m.RetailTypical > 0 {
		m.DiffPercent = percentOf(m.CustomerPrice-m.RetailTypical, m.RetailTypical)
	}
	return m
}

// monthTotals ยอดของ 1 ช่วง สำหรับกราฟย้อนหลัง / ช่วงก่อนหน้า (คิดกำไรด้วย transaction.NewSummary)
func monthTotals(period utils.DateRange, bills int, a transaction.Amounts) MonthTotals {
	s := transaction.NewSummary(period, a)
	return MonthTotals{
		Month:            period.From[:7], // "2026-10-17" → "2026-10"
		Income:           s.Income,
		Expense:          s.Expense,
		Profit:           s.Profit,
		NetSales:         s.Sales,
		Bills:            bills,
		GrossProfit:      s.GrossProfit,
		Revenue:          s.Revenue,
		CostOfSales:      s.CostOfSales,
		OperatingExpense: s.OperatingExpense,
	}
}

// trendMonths เดือนย้อนหลัง count เดือน (ทั้งเดือน) เรียงเก่า → ใหม่
// เดือนสุดท้าย = เดือนของวันสุดท้ายที่เลือก เช่น เลือก 1–16 พ.ย. 2026 → มิ.ย. ถึง พ.ย. 2026
func trendMonths(lastDay string, count int) []utils.DateRange {
	end, err := time.Parse(utils.DateLayout, lastDay)
	if err != nil {
		return []utils.DateRange{}
	}
	firstOfMonth := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
	months := make([]utils.DateRange, 0, count)
	for i := count - 1; i >= 0; i-- {
		months = append(months, utils.MonthRange(firstOfMonth.AddDate(0, -i, 0)))
	}
	return months
}
