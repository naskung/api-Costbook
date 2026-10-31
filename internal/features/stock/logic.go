package stock

import "github.com/naskung/api-Costbook/internal/utils"

// averageCost ต้นทุนเฉลี่ยใหม่หลังรับของเข้า
// = (ของเดิม × ทุนเดิม + ของใหม่ × ทุนใหม่) ÷ จำนวนรวม
// เช่น เดิม 10 ขวด ทุน 5 บาท + รับเพิ่ม 10 ขวด ทุน 6 บาท → ทุนเฉลี่ย 5.50 บาท
// ของเดิมหมด (หรือติดลบ) → ใช้ทุนใหม่เลย
func averageCost(oldStock int, oldCost float64, quantity int, unitCost float64) float64 {
	if oldStock <= 0 {
		return utils.Round2(unitCost)
	}
	total := float64(oldStock)*oldCost + float64(quantity)*unitCost
	return utils.Round2(total / float64(oldStock+quantity))
}
