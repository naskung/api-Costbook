package utils

// CostPerUnit ต้นทุนต่อชิ้น = (วัตถุดิบ + ค่าแรง + ค่าใช้จ่ายอื่น) ÷ จำนวนที่ผลิตได้ต่อรอบ
// จำนวนผลิตเป็น 0 หรือติดลบ → คืน 0 (กันหารด้วย 0)
func CostPerUnit(material, labor, overhead float64, quantity int) float64 {
	if quantity <= 0 {
		return 0
	}
	return Round2((material + labor + overhead) / float64(quantity))
}
