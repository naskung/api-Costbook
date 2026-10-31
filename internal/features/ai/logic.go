package ai

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/naskung/api-Costbook/internal/features/company"
	"github.com/naskung/api-Costbook/internal/features/cost"
	"github.com/naskung/api-Costbook/internal/features/field"
	"github.com/naskung/api-Costbook/internal/features/product"
	"github.com/naskung/api-Costbook/internal/utils"
)

// ขีดจำกัดของผลเช็คราคาตลาดที่เก็บลง DB
const (
	maxReportItems  = 10
	maxReportText   = 2000
	maxSourceText   = 300
	confidenceLow   = "low"
	confidenceMid   = "medium"
	confidenceHigh  = "high"
	sourceRetail    = "retail"
	sourceWholesale = "wholesale"
	sourceMaterial  = "material"
)

// unitOf หน่วยขายของสินค้า (ไม่ได้กรอก → "ชิ้น")
func unitOf(p product.Product) string {
	if p.Unit == "" {
		return "ชิ้น"
	}
	return p.Unit
}

// vatNote ราคาขายของร้านรวม VAT หรือไม่ (ให้ AI เทียบกับราคาตลาดถูก)
func vatNote(shop company.Company) string {
	if !shop.VATRegistered {
		return "ร้านไม่จด VAT ราคานี้คือราคาที่ลูกค้าจ่าย"
	}
	if shop.PricesIncludeVAT {
		return "รวม VAT แล้ว ราคานี้คือราคาที่ลูกค้าจ่าย"
	}
	return "ยังไม่รวม VAT 7% ลูกค้าจ่ายจริง = ราคานี้ × 1.07"
}

// fieldValues ช่องข้อมูลเพิ่มเติมของสินค้าแบบ "ชื่อช่อง: ค่า" เรียงตามลำดับช่องในหน้าตั้งค่า (ไม่มีค่าไม่ใส่)
func fieldValues(p product.Product, fields []field.ProductField) []string {
	values := []string{}
	for _, f := range fields {
		value := strings.TrimSpace(p.CustomFields[f.ID])
		if value != "" {
			values = append(values, f.Name+": "+value)
		}
	}
	return values
}

// researchPrompt ข้อมูลสินค้าที่ส่งให้ AI ไปเช็คราคาตลาด
func researchPrompt(p product.Product, sheet cost.CostSheet, fields []field.ProductField, shop company.Company) string {
	unit := unitOf(p)
	var b strings.Builder
	b.WriteString("สินค้าที่ต้องเช็คราคาตลาด\n")
	fmt.Fprintf(&b, "- ชื่อ: %s\n", p.Name)
	if p.Category != "" {
		fmt.Fprintf(&b, "- หมวด: %s\n", p.Category)
	}
	if p.Description != "" {
		fmt.Fprintf(&b, "- รายละเอียด: %s\n", p.Description)
	}
	for _, value := range fieldValues(p, fields) {
		fmt.Fprintf(&b, "- %s\n", value)
	}
	fmt.Fprintf(&b, "- หน่วยขาย: %s\n", unit)
	fmt.Fprintf(&b, "- ต้นทุนของร้าน: %.2f บาท/%s\n", p.CostPrice, unit)
	fmt.Fprintf(&b, "- ราคาขายของร้าน: %.2f บาท/%s (%s)\n", p.SellPrice, unit, vatNote(shop))

	if len(sheet.Items) > 0 {
		fmt.Fprintf(&b, "\nรายการต้นทุนของร้าน ต่อการผลิต 1 รอบ (ได้ %d %s):\n", sheet.BatchQuantity, unit)
		for _, item := range sheet.Items {
			fmt.Fprintf(&b, "- %s: %.2f บาท\n", costItemText(item), item.Amount)
		}
	}
	return b.String()
}

// costItemText บรรยายบรรทัดต้นทุน เช่น "เส้น PLA 160 กรัม × 0.45" หรือ "เผื่อพิมพ์เสีย 10%"
func costItemText(item cost.CostItem) string {
	if item.Kind == "percent" {
		return fmt.Sprintf("%s %g%% ของต้นทุนอื่น", item.Name, item.Percent)
	}
	return strings.TrimSpace(fmt.Sprintf("%s %g %s × %g", item.Name, item.Quantity, item.Unit, item.UnitPrice))
}

// cleanReport ทำผลจาก AI ให้เก็บลง DB ได้ปลอดภัย:
// ตัวเลขติดลบ → 0, ต่ำสุด > สูงสุด → สลับ, ปัดเศษ 2 ตำแหน่ง, ตัดข้อความยาวเกิน, ตัดลิงก์ที่ไม่ใช่ http(s)
func cleanReport(r MarketReport) MarketReport {
	r.RetailLow, r.RetailHigh = priceRange(r.RetailLow, r.RetailHigh)
	r.RetailTypical = money(r.RetailTypical)
	r.WholesaleLow, r.WholesaleHigh = priceRange(r.WholesaleLow, r.WholesaleHigh)
	r.MarketCost = money(r.MarketCost)
	r.SuggestedPrice = money(r.SuggestedPrice)
	r.Summary = shortText(r.Summary, maxReportText)
	r.Advice = shortText(r.Advice, maxReportText)
	r.WholesaleNote = shortText(r.WholesaleNote, maxSourceText)
	if r.Confidence != confidenceMid && r.Confidence != confidenceHigh {
		r.Confidence = confidenceLow
	}

	items := []MarketCostItem{}
	for _, item := range r.CostItems {
		if strings.TrimSpace(item.Name) == "" || len(items) == maxReportItems {
			continue
		}
		items = append(items, MarketCostItem{
			Name:   shortText(item.Name, maxSourceText),
			Amount: money(item.Amount),
			Note:   shortText(item.Note, maxSourceText),
		})
	}
	r.CostItems = items

	sources := []MarketSource{}
	for _, s := range r.Sources {
		isLink := strings.HasPrefix(s.URL, "https://") || strings.HasPrefix(s.URL, "http://")
		if !isLink || len(sources) == maxReportItems {
			continue
		}
		if s.Kind != sourceWholesale && s.Kind != sourceMaterial {
			s.Kind = sourceRetail
		}
		sources = append(sources, MarketSource{
			Title: shortText(s.Title, maxSourceText),
			URL:   s.URL,
			Price: money(s.Price),
			Kind:  s.Kind,
		})
	}
	r.Sources = sources
	return r
}

// money เงินห้ามติดลบ + ปัดเศษ 2 ตำแหน่ง
func money(value float64) float64 {
	if value < 0 {
		return 0
	}
	return utils.Round2(value)
}

// priceRange ช่วงราคา (ต่ำสุด, สูงสุด) ถ้าใส่กลับด้านมาให้สลับ
func priceRange(low, high float64) (float64, float64) {
	low, high = money(low), money(high)
	if high > 0 && low > high {
		return high, low
	}
	return low, high
}

// shortText ตัดช่องว่างหน้า-หลัง และตัดให้ยาวไม่เกิน max ตัวอักษร
func shortText(text string, max int) string {
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) <= max {
		return text
	}
	return string([]rune(text)[:max])
}
