package company

import (
	"strings"

	"github.com/naskung/api-Costbook/internal/utils"
)

// Company ข้อมูลร้าน (ผู้ขาย) + รูปแบบร้าน ใช้บนหน้าเว็บและในเอกสารขาย
// has_logo = true แปลว่ามีโลโก้ → Flutter โหลดรูปจาก GET /api/v1/company/logo
// vat_registered = false → ออกใบเสร็จรับเงิน ไม่คิด VAT / prices_include_vat = true → ราคาขายรวม VAT แล้ว
// use_cost_page = false → ซ่อนหน้าคำนวณต้นทุน (ร้านซื้อมาขายไป)
// track_delivery = true → เอกสารขายมีสถานะการส่งสินค้า (รอส่ง / ส่งแล้ว / ได้รับแล้ว) ใบใหม่เริ่มที่รอส่ง
type Company struct {
	Name             string `json:"name"`
	Address          string `json:"address"`
	TaxID            string `json:"tax_id"`
	Branch           string `json:"branch"`
	Phone            string `json:"phone"`
	HasLogo          bool   `json:"has_logo"`
	VATRegistered    bool   `json:"vat_registered"`
	PricesIncludeVAT bool   `json:"prices_include_vat"`
	UseCostPage      bool   `json:"use_cost_page"`
	TrackDelivery    bool   `json:"track_delivery"`
}

// CompanyRequest ข้อมูลที่รับมาตอนแก้ข้อมูลร้าน
// ข้อความ: ส่งมาครบทุกช่อง ช่องที่ไม่ส่งจะกลายเป็นค่าว่าง
// รูปแบบร้าน (4 ช่องท้าย) เป็น pointer: ไม่ส่งมา (nil) = ใช้ค่าเดิม → แก้แค่ชื่อร้าน รูปแบบร้านไม่เปลี่ยนเอง
type CompanyRequest struct {
	Name             string `json:"name"`
	Address          string `json:"address"`
	TaxID            string `json:"tax_id"`
	Branch           string `json:"branch"`
	Phone            string `json:"phone"`
	VATRegistered    *bool  `json:"vat_registered"`
	PricesIncludeVAT *bool  `json:"prices_include_vat"`
	UseCostPage      *bool  `json:"use_cost_page"`
	TrackDelivery    *bool  `json:"track_delivery"`
}

// Logo ไฟล์รูปโลโก้ (ส่งออกเป็นรูปตรง ๆ ไม่ได้ส่งเป็น JSON)
type Logo struct {
	Data        []byte
	ContentType string
}

// Validate เช็คข้อมูลก่อนบันทึก คืน error code ("" = ผ่าน)
func (req CompanyRequest) Validate() string {
	if strings.TrimSpace(req.Name) == "" {
		return "company_name_required"
	}
	if req.TaxID != "" && !utils.IsTaxID(req.TaxID) {
		return "company_tax_id_invalid"
	}
	return ""
}
