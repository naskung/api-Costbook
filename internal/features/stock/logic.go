package stock

import (
	"math"

	"github.com/naskung/api-Costbook/internal/utils"
)

// materialNeed ต้องใช้วัสดุเท่าไหร่ (หน่วยของวัสดุ) ผลิต quantity ชิ้น
// ใบต้นทุนบอกว่า 1 รอบ (batchQuantity ชิ้น) ใช้ perBatch → ใช้ perBatch ÷ batchQuantity ต่อชิ้น ปัดเป็นจำนวนเต็มตอนรวมแล้ว
// เช่น พวงกุญแจ 6 ชิ้น/รอบ ใช้ PLA 60 กรัม → ผลิต 4 ชิ้น ใช้ 40 กรัม
func materialNeed(perBatch float64, batchQuantity, quantity int) int {
	if batchQuantity <= 0 {
		batchQuantity = 1
	}
	return int(math.Round(perBatch / float64(batchQuantity) * float64(quantity)))
}

// withSaleProfit ใส่กำไรของแถวที่ขาย จากยอดของสินค้านี้ในบิล (lineQuantity / lineNet / lineCost)
// แบ่งตามจำนวนของแถวนี้ — ใบเดียวมีสินค้าเดียวกันหลายบรรทัดได้ (ตัดสต็อกบรรทัดละแถว) แต่ละแถวได้ส่วนของตัวเอง
// lineQuantity = 0 → ไม่ใช่แถวที่ขาย (รับเข้า / เบิกเอง / วัสดุ / บิลที่ยกเลิกแล้ว) ไม่มีกำไร
// เช่น ขาย 2 ชิ้น ยอดก่อน VAT 250.52 ทุนชิ้นละ 113.87 → กำไร 22.78
func withSaleProfit(m Movement, lineQuantity int, lineNet, lineCost float64) Movement {
	if lineQuantity <= 0 {
		return m
	}
	share := float64(m.Quantity) / float64(lineQuantity)
	m.HasProfit = true
	m.SaleAmount = utils.Round2(lineNet * share)
	m.CostAmount = utils.Round2(lineCost * share)
	m.Profit = utils.Round2(m.SaleAmount - m.CostAmount)
	return m
}

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
