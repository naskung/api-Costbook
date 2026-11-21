package quotation

import (
	"fmt"

	"github.com/naskung/api-Costbook/internal/utils"
)

// quotationNumber เลขที่ใบเสนอราคา เช่น ปี 2026 ใบที่ 1 → "QT-2026-0001" (รันใหม่ทุกปี)
func quotationNumber(year, number int) string {
	return fmt.Sprintf("QT-%d-%04d", year, number)
}

// itemsTotal รวมราคาสินค้าทุกบรรทัด
func itemsTotal(items []ItemRequest) float64 {
	var total float64
	for _, item := range items {
		total += utils.LineAmount(item.Quantity, item.UnitPrice)
	}
	return utils.Round2(total)
}
