package utils

// DocAmounts ยอดเงินของเอกสาร 1 ใบ (ใบกำกับภาษี / ใบเสร็จรับเงิน / ใบเสนอราคา)
type DocAmounts struct {
	ItemsTotal float64 // รวมราคาสินค้า
	Discount   float64 // ส่วนลดทั้งบิล
	Subtotal   float64 // มูลค่าก่อน VAT
	VAT        float64
	Total      float64 // ยอดสุทธิที่ลูกค้าจ่าย
}

// LineAmount ยอดเงินของสินค้า 1 บรรทัด = จำนวน × ราคาต่อหน่วย
func LineAmount(quantity int, unitPrice float64) float64 {
	return Round2(float64(quantity) * unitPrice)
}

// DocumentAmounts คิดยอดเงินของเอกสาร (VAT คิดจากยอดรวมทั้งใบหลังหักส่วนลด ไม่คิดทีละบรรทัด)
//   - ไม่คิด VAT (ร้านไม่จด VAT): ยอดสุทธิ = รวม − ส่วนลด
//   - ราคาไม่รวม VAT: มูลค่าก่อน VAT = รวม − ส่วนลด แล้วบวก VAT 7%
//   - ราคารวม VAT แล้ว: ยอดสุทธิ = รวม − ส่วนลด แล้วถอด VAT ออกมาแสดง
func DocumentAmounts(itemsTotal, discount float64, withVAT, pricesIncludeVAT bool) DocAmounts {
	m := DocAmounts{ItemsTotal: Round2(itemsTotal), Discount: Round2(discount)}
	afterDiscount := Round2(m.ItemsTotal - m.Discount)

	switch {
	case !withVAT:
		m.Subtotal = afterDiscount
		m.Total = afterDiscount
	case pricesIncludeVAT:
		m.Total = afterDiscount
		m.Subtotal = PriceBeforeVAT(afterDiscount)
		m.VAT = Round2(m.Total - m.Subtotal)
	default:
		m.Subtotal = afterDiscount
		m.VAT = VAT(afterDiscount)
		m.Total = Round2(m.Subtotal + m.VAT)
	}
	return m
}
