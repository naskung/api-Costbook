package company

import (
	"strings"

	"github.com/naskung/api-Costbook/internal/utils"
)

// Company ข้อมูลร้าน (ผู้ขาย) ใช้บนหน้าเว็บและใบกำกับภาษี
// has_logo = true แปลว่ามีโลโก้ → Flutter โหลดรูปจาก GET /api/v1/company/logo
type Company struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	TaxID   string `json:"tax_id"`
	Branch  string `json:"branch"`
	Phone   string `json:"phone"`
	HasLogo bool   `json:"has_logo"`
}

// CompanyRequest ข้อมูลที่รับมาตอนแก้ข้อมูลร้าน (ส่งมาครบทุกช่อง ช่องที่ไม่ส่งจะกลายเป็นค่าว่าง)
type CompanyRequest struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	TaxID   string `json:"tax_id"`
	Branch  string `json:"branch"`
	Phone   string `json:"phone"`
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
