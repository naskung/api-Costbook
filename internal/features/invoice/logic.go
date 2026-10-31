package invoice

import (
	"fmt"

	"github.com/naskung/api-Costbook/internal/utils"
)

// invoiceNumber เลขที่ใบกำกับ เช่น ปี 2026 ใบที่ 1 → "INV-2026-0001" (รันใหม่ทุกปี)
func invoiceNumber(year, number int) string {
	return fmt.Sprintf("INV-%d-%04d", year, number)
}

// lineAmount ยอดเงินของสินค้า 1 บรรทัด = จำนวน × ราคาต่อหน่วย
func lineAmount(quantity int, unitPrice float64) float64 {
	return utils.Round2(float64(quantity) * unitPrice)
}

// totals ยอดรวมก่อน VAT, VAT 7% (คิดจากยอดรวมทั้งใบครั้งเดียว) และยอดรวมสุทธิ
func totals(items []ItemRequest) (subtotal, vat, total float64) {
	for _, item := range items {
		subtotal += lineAmount(item.Quantity, item.UnitPrice)
	}
	subtotal = utils.Round2(subtotal)
	vat = utils.VAT(subtotal)
	total = utils.Round2(subtotal + vat)
	return subtotal, vat, total
}
