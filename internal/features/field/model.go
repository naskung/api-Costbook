package field

import (
	"slices"
	"strings"

	"github.com/naskung/api-Costbook/internal/utils"
)

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

// FieldRequest ข้อมูลที่รับมาตอนเพิ่ม / แก้ช่อง — ไม่มีตำแหน่ง (เพิ่มใหม่อยู่ท้ายฟอร์ม / ย้ายใช้ PUT /product-form-layout)
type FieldRequest struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Options    []string `json:"options"`
	ShowInList bool     `json:"show_in_list"`
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

// ช่องหลักของฟอร์มเพิ่ม / แก้สินค้า (ร้านย้ายตำแหน่งได้ แต่ลบไม่ได้) — ต้องตรงกับ CHECK ใน migration 000017
// code_category = รหัส + หมวดหมู่ (อยู่คู่กันในแถวเดียว), prices = ต้นทุน + ราคาขาย, sale_stock = สถานะขาย / นับสต็อก / ขั้นต่ำ
const (
	BlockImage        = "image"
	BlockName         = "name"
	BlockCodeCategory = "code_category"
	BlockUnit         = "unit"
	BlockPrices       = "prices"
	BlockSaleStock    = "sale_stock"
	BlockDescription  = "description"
)

// FormBlocks ช่องหลักทั้งหมด เรียงตามค่าเริ่มต้น
var FormBlocks = []string{BlockImage, BlockName, BlockCodeCategory, BlockUnit, BlockPrices, BlockSaleStock, BlockDescription}

// LayoutItem ตำแหน่งของ 1 ช่องในฟอร์มสินค้า
// key = ชื่อช่องหลัก (image, name, ...) หรือ id ของช่องที่ร้านเพิ่มเอง / sort_order น้อย = อยู่บน (ใช้เลขชุดเดียวกันทั้งสองแบบ)
type LayoutItem struct {
	Key       string `json:"key"`
	SortOrder int    `json:"sort_order"`
}

// LayoutRequest ลำดับใหม่ของทั้งฟอร์ม (PUT /product-form-layout) — ต้องส่งครบทุกช่อง
type LayoutRequest struct {
	Items []LayoutItem `json:"items"`
}

// Validate เช็คก่อนถาม DB: key ไม่ซ้ำ, เป็นช่องหลักหรือ id (UUID), sort_order ไม่ติดลบ, มีช่องหลักครบ
// (ส่งช่องที่ร้านเพิ่มครบไหม repository เช็คกับ DB อีกที)
func (req LayoutRequest) Validate() string {
	seen := map[string]bool{}
	for _, item := range req.Items {
		isKnown := slices.Contains(FormBlocks, item.Key) || utils.IsUUID(item.Key)
		if !isKnown || seen[item.Key] || item.SortOrder < 0 {
			return "form_layout_invalid"
		}
		seen[item.Key] = true
	}
	for _, block := range FormBlocks {
		if !seen[block] {
			return "form_layout_invalid"
		}
	}
	return ""
}
