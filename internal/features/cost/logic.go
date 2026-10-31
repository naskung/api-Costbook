package cost

import "github.com/naskung/api-Costbook/internal/utils"

// costPrice รวมต้นทุนแต่ละประเภท แล้วหารด้วยจำนวนที่ผลิตได้ต่อรอบ = ต้นทุนต่อชิ้น
func costPrice(req CostSheetRequest) float64 {
	var material, labor, overhead float64
	for _, item := range req.Items {
		switch item.Type {
		case "material":
			material += item.Amount
		case "labor":
			labor += item.Amount
		case "overhead":
			overhead += item.Amount
		}
	}
	return utils.CostPerUnit(material, labor, overhead, req.BatchQuantity)
}
