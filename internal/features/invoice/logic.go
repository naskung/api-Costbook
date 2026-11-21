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

// สถานะการส่งสินค้า (ใช้เมื่อร้านเปิดติดตามการส่งในหน้าตั้งค่า)
const (
	deliveryPending  = "pending"  // รอส่ง
	deliveryShipped  = "shipped"  // ส่งแล้ว
	deliveryReceived = "received" // ลูกค้าได้รับแล้ว
)

// docTypeFor ร้านจด VAT → ใบกำกับภาษี, ไม่จด → ใบเสร็จรับเงิน
func docTypeFor(vatRegistered bool) string {
	if vatRegistered {
		return docTaxInvoice
	}
	return docReceipt
}

// deliveryStatusFor สถานะการส่งของใบใหม่: ร้านติดตามการส่ง → รอส่ง / ไม่ติดตาม → ถือว่าลูกค้าได้ของไปเลย
func deliveryStatusFor(trackDelivery bool) string {
	if trackDelivery {
		return deliveryPending
	}
	return deliveryReceived
}

// isDeliveryStatus เช็คว่าเป็นสถานะการส่งที่ใช้ได้
func isDeliveryStatus(status string) bool {
	return status == deliveryPending || status == deliveryShipped || status == deliveryReceived
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

// itemsTotal รวมราคาสินค้าทุกบรรทัด
func itemsTotal(items []ItemRequest) float64 {
	var total float64
	for _, item := range items {
		total += utils.LineAmount(item.Quantity, item.UnitPrice)
	}
	return utils.Round2(total)
}
