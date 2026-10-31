package invoice

import (
	"fmt"

	"github.com/naskung/api-Costbook/internal/utils"
)

// ชนิดเอกสารขาย
const (
	docTaxInvoice = "tax_invoice" // ใบกำกับภาษี (ร้านจด VAT)
	docReceipt    = "receipt"     // ใบเสร็จรับเงิน (ร้านไม่จด VAT)
)

// money ยอดเงินของเอกสาร 1 ใบ
type money struct {
	itemsTotal float64 // รวมราคาสินค้า
	discount   float64 // ส่วนลดทั้งบิล
	subtotal   float64 // มูลค่าก่อน VAT
	vat        float64
	total      float64 // ยอดสุทธิที่ลูกค้าจ่าย
}

// docTypeFor ร้านจด VAT → ใบกำกับภาษี, ไม่จด → ใบเสร็จรับเงิน
func docTypeFor(vatRegistered bool) string {
	if vatRegistered {
		return docTaxInvoice
	}
	return docReceipt
}

// docPrefix ตัวนำหน้าเลขที่เอกสาร: ใบกำกับ = INV, ใบเสร็จ = RC
func docPrefix(docType string) string {
	if docType == docReceipt {
		return "RC"
	}
	return "INV"
}

// invoiceNumber เลขที่เอกสาร เช่น ใบกำกับปี 2026 ใบที่ 1 → "INV-2026-0001" (รันใหม่ทุกปี แยกตามชนิดเอกสาร)
func invoiceNumber(prefix string, year, number int) string {
	return fmt.Sprintf("%s-%d-%04d", prefix, year, number)
}

// lineAmount ยอดเงินของสินค้า 1 บรรทัด = จำนวน × ราคาต่อหน่วย
func lineAmount(quantity int, unitPrice float64) float64 {
	return utils.Round2(float64(quantity) * unitPrice)
}

// itemsTotal รวมราคาสินค้าทุกบรรทัด
func itemsTotal(items []ItemRequest) float64 {
	var total float64
	for _, item := range items {
		total += lineAmount(item.Quantity, item.UnitPrice)
	}
	return utils.Round2(total)
}

// documentAmounts คิดยอดเงินของเอกสาร (VAT คิดจากยอดรวมทั้งใบหลังหักส่วนลด ไม่คิดทีละบรรทัด)
//   - ใบเสร็จ (ไม่จด VAT): ไม่มี VAT ยอดสุทธิ = รวม − ส่วนลด
//   - ใบกำกับ ราคาไม่รวม VAT: มูลค่าก่อน VAT = รวม − ส่วนลด แล้วบวก VAT 7%
//   - ใบกำกับ ราคารวม VAT แล้ว: ยอดสุทธิ = รวม − ส่วนลด แล้วถอด VAT ออกมาแสดง
func documentAmounts(items []ItemRequest, discount float64, docType string, pricesIncludeVAT bool) money {
	m := money{itemsTotal: itemsTotal(items), discount: utils.Round2(discount)}
	afterDiscount := utils.Round2(m.itemsTotal - m.discount)

	switch {
	case docType == docReceipt:
		m.subtotal = afterDiscount
		m.total = afterDiscount
	case pricesIncludeVAT:
		m.total = afterDiscount
		m.subtotal = utils.PriceBeforeVAT(afterDiscount)
		m.vat = utils.Round2(m.total - m.subtotal)
	default:
		m.subtotal = afterDiscount
		m.vat = utils.VAT(afterDiscount)
		m.total = utils.Round2(m.subtotal + m.vat)
	}
	return m
}
