package cost

import "github.com/naskung/api-Costbook/internal/utils"

// itemAmounts จำนวนเงินของแต่ละบรรทัด (เรียงตาม items)
// fixed = จำนวน × ราคาต่อหน่วย, percent = % ของรวมบรรทัด fixed ทั้งหมด (เช่น เผื่อพิมพ์เสีย 10%)
func itemAmounts(items []CostItemRequest) []float64 {
	amounts := make([]float64, len(items))
	var fixedTotal float64
	for i, item := range items {
		if item.Kind == "fixed" {
			amounts[i] = utils.Round2(item.Quantity * item.UnitPrice)
			fixedTotal += amounts[i]
		}
	}
	for i, item := range items {
		if item.Kind == "percent" {
			amounts[i] = utils.Round2(fixedTotal * item.Percent / 100)
		}
	}
	return amounts
}

// costPrice รวมเงินแต่ละประเภท แล้วหารด้วยจำนวนที่ผลิตได้ต่อรอบ = ต้นทุนต่อชิ้น
func costPrice(items []CostItemRequest, amounts []float64, batchQuantity int) float64 {
	var material, labor, overhead float64
	for i, item := range items {
		switch item.Type {
		case "material":
			material += amounts[i]
		case "labor":
			labor += amounts[i]
		case "overhead":
			overhead += amounts[i]
		}
	}
	return utils.CostPerUnit(material, labor, overhead, batchQuantity)
}
