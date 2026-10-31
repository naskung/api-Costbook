package utils

// VATPercent อัตรา VAT ของไทย 7%
const VATPercent = 7

// VAT คิด VAT จากราคาก่อน VAT เช่น 100 → 7
// ใบกำกับภาษีคิดจากยอดรวมทั้งใบครั้งเดียว ไม่คิดทีละบรรทัด
func VAT(priceBeforeVAT float64) float64 {
	return Round2(priceBeforeVAT * VATPercent / 100)
}

// PriceBeforeVAT ถอด VAT ออกจากราคาที่รวม VAT แล้ว เช่น 107 → 100
// ใช้กับร้านที่ตั้งราคาขายแบบรวม VAT
func PriceBeforeVAT(totalWithVAT float64) float64 {
	return Round2(totalWithVAT * 100 / (100 + VATPercent))
}
