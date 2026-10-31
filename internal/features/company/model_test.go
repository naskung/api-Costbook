package company

import "testing"

func TestCompanyRequest_Validate(t *testing.T) {
	tests := []struct {
		name string
		req  CompanyRequest
		want string
	}{
		{"ข้อมูลครบ ผ่าน", CompanyRequest{Name: "ร้านตัวอย่าง", TaxID: "0105555555555", Branch: "สำนักงานใหญ่"}, ""},
		{"ไม่กรอกเลขภาษี ผ่าน", CompanyRequest{Name: "ร้านตัวอย่าง"}, ""},
		{"ชื่อว่าง ไม่ผ่าน", CompanyRequest{Name: ""}, "company_name_required"},
		{"ชื่อมีแต่ช่องว่าง ไม่ผ่าน", CompanyRequest{Name: "   "}, "company_name_required"},
		{"เลขภาษี 12 หลัก ไม่ผ่าน", CompanyRequest{Name: "ร้านตัวอย่าง", TaxID: "010555555555"}, "company_tax_id_invalid"},
		{"เลขภาษี 14 หลัก ไม่ผ่าน", CompanyRequest{Name: "ร้านตัวอย่าง", TaxID: "01055555555555"}, "company_tax_id_invalid"},
		{"เลขภาษีมีตัวอักษร ไม่ผ่าน", CompanyRequest{Name: "ร้านตัวอย่าง", TaxID: "010555555555A"}, "company_tax_id_invalid"},
		{"เลขภาษีมีขีด ไม่ผ่าน", CompanyRequest{Name: "ร้านตัวอย่าง", TaxID: "0-1055-55555-55"}, "company_tax_id_invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.req.Validate()
			if got != tt.want {
				t.Errorf("Validate() = %q, ต้องได้ %q", got, tt.want)
			}
		})
	}
}
