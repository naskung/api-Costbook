package field

import "strings"

// ProductField ช่องข้อมูลเพิ่มเติมของสินค้าที่ร้านตั้งเอง เช่น วัสดุ, สี, วันหมดอายุ, ยี่ห้อ
// type: text = ข้อความ, number = ตัวเลข, date = วันที่, select = เลือกจาก options
// show_in_list = true → แสดงเป็น column ในตารางสินค้า
// ค่าที่กรอกของแต่ละสินค้าเก็บใน products.custom_fields {"<id>": "ค่า"}
type ProductField struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Options    []string `json:"options"`
	ShowInList bool     `json:"show_in_list"`
	SortOrder  int      `json:"sort_order"`
}

// FieldRequest ข้อมูลที่รับมาตอนเพิ่ม / แก้ช่อง
type FieldRequest struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Options    []string `json:"options"`
	ShowInList bool     `json:"show_in_list"`
	SortOrder  int      `json:"sort_order"`
}

// Validate เช็คข้อมูลก่อนบันทึก คืน error code ("" = ผ่าน)
func (req FieldRequest) Validate() string {
	if strings.TrimSpace(req.Name) == "" {
		return "field_name_required"
	}
	if req.Type != "text" && req.Type != "number" && req.Type != "date" && req.Type != "select" {
		return "field_type_invalid"
	}
	if req.Type == "select" && len(req.Options) == 0 {
		return "field_options_required"
	}
	for _, option := range req.Options {
		if strings.TrimSpace(option) == "" {
			return "field_options_required"
		}
	}
	return ""
}

// options ที่จะบันทึก: ช่องที่ไม่ใช่ select ไม่เก็บ options / ไม่ส่งมา = รายการว่าง (JSON ได้ [] ไม่ใช่ null)
func (req FieldRequest) cleanOptions() []string {
	if req.Type != "select" || req.Options == nil {
		return []string{}
	}
	return req.Options
}
