package product

import (
	"strings"

	"github.com/naskung/api-Costbook/internal/utils"
)

// ความยาวสูงสุดของช่องเพิ่มเติม กันข้อมูลใหญ่ผิดปกติ
const (
	maxCustomFields     = 50
	maxCustomFieldValue = 500
)

// Product ข้อมูลสินค้า (ชื่อ json ต้องตรงกับ ProductModel ฝั่ง Flutter)
// stock เปลี่ยนผ่านหน้าสต็อก / เอกสารขายเท่านั้น, batch_quantity แก้ในหน้าคำนวณต้นทุน
// track_stock = false → งานสั่งทำ / ค่าบริการ ไม่นับสต็อก ขายได้ไม่จำกัด
// for_sale = false → วัสดุ / ส่วนประกอบ ไม่ขาย (ไม่ขึ้นในหน้าขาย / เอกสารขาย) ใช้ในใบต้นทุนของสินค้าอื่น
// custom_fields = ค่าของช่องเพิ่มเติมที่ร้านตั้งเอง {"<id ช่อง>": "ค่า"} (ดู /product-fields)
// category_id = หมวด (ว่าง = ไม่มีหมวด, จัดการที่ /product-categories), category = ชื่อหมวด (อ่านอย่างเดียว ไว้แสดง)
// image_version = เวลาที่เปลี่ยนรูปล่าสุด (0 = ไม่มีรูป) → รูปอยู่ที่ GET /products/{id}/image?v=<image_version>
type Product struct {
	ID            string            `json:"id"`
	Code          string            `json:"code"`
	Name          string            `json:"name"`
	CategoryID    string            `json:"category_id"`
	Category      string            `json:"category"`
	Description   string            `json:"description"`
	Unit          string            `json:"unit"`
	CostPrice     float64           `json:"cost_price"`
	SellPrice     float64           `json:"sell_price"`
	Stock         int               `json:"stock"`
	MinStock      int               `json:"min_stock"`
	BatchQuantity int               `json:"batch_quantity"`
	TrackStock    bool              `json:"track_stock"`
	ForSale       bool              `json:"for_sale"`
	CustomFields  map[string]string `json:"custom_fields"`
	ImageVersion  int64             `json:"image_version"`
}

// Image รูปสินค้า (ไฟล์ + ชนิดรูป เช่น image/png)
type Image struct {
	Data        []byte
	ContentType string
}

// ProductRequest ข้อมูลที่รับมาตอนเพิ่ม / แก้สินค้า
// track_stock เป็น pointer เพื่อแยก "ไม่ส่งมา" ออกจาก "ส่ง false มา"
// ไม่ส่ง track_stock / for_sale / custom_fields / category_id มา: ตอนเพิ่ม = นับสต็อก / ขาย / ไม่มีช่องเพิ่มเติม / ไม่มีหมวด, ตอนแก้ = ใช้ค่าเดิม
// category_id = "" → ไม่มีหมวด
type ProductRequest struct {
	Code         string            `json:"code"`
	Name         string            `json:"name"`
	CategoryID   *string           `json:"category_id"`
	Description  string            `json:"description"`
	Unit         string            `json:"unit"`
	CostPrice    float64           `json:"cost_price"`
	SellPrice    float64           `json:"sell_price"`
	MinStock     int               `json:"min_stock"`
	TrackStock   *bool             `json:"track_stock"`
	ForSale      *bool             `json:"for_sale"`
	CustomFields map[string]string `json:"custom_fields"`
}

// Validate เช็คข้อมูลก่อนบันทึก คืน error code ("" = ผ่าน)
func (req ProductRequest) Validate() string {
	if strings.TrimSpace(req.Name) == "" {
		return "product_name_required"
	}
	if req.CostPrice < 0 || req.SellPrice < 0 {
		return "product_price_invalid"
	}
	if req.MinStock < 0 {
		return "product_min_stock_invalid"
	}
	if req.CategoryID != nil && *req.CategoryID != "" && !utils.IsUUID(*req.CategoryID) {
		return "product_category_invalid"
	}
	if len(req.CustomFields) > maxCustomFields {
		return "product_custom_fields_invalid"
	}
	for _, value := range req.CustomFields {
		if len(value) > maxCustomFieldValue {
			return "product_custom_fields_invalid"
		}
	}
	return ""
}

// categoryIDText ค่าตอนเพิ่มสินค้า: ไม่ส่งมา = ไม่มีหมวด ("")
func (req ProductRequest) categoryIDText() string {
	if req.CategoryID == nil {
		return ""
	}
	return *req.CategoryID
}

// forSale ค่าตอนเพิ่มสินค้า: ไม่ส่ง for_sale มา → ขาย (ค่าเริ่มต้น)
func (req ProductRequest) forSale() bool {
	if req.ForSale == nil {
		return true
	}
	return *req.ForSale
}

// trackStock ค่าตอนเพิ่มสินค้า: ไม่ส่ง track_stock มา → นับสต็อก (ค่าเริ่มต้น)
func (req ProductRequest) trackStock() bool {
	if req.TrackStock == nil {
		return true
	}
	return *req.TrackStock
}

// customFields ค่าตอนเพิ่มสินค้า: ไม่ส่งมา → {} (column ใน DB ห้ามเป็น null)
func (req ProductRequest) customFields() map[string]string {
	if req.CustomFields == nil {
		return map[string]string{}
	}
	return req.CustomFields
}

// kind ของรายการสินค้า (?kind=) — sale = สินค้าที่ขาย / material = วัสดุ (ไม่ขาย) / ว่าง = ทั้งหมด
const (
	KindForSale  = "sale"
	KindMaterial = "material"
)

// ListFilter ตัวกรองของรายการสินค้า (ค่าว่าง = ไม่กรอง / Limit 0 = ทั้งหมด)
// Query ค้นจาก ชื่อ / รหัส / หมวด / รายละเอียด / ค่าในช่องข้อมูลเพิ่มเติม (ไม่สนตัวพิมพ์เล็ก-ใหญ่ เหมือนที่หน้าเว็บเคยค้นเอง)
// OnlyLow = เฉพาะใกล้หมด (นับสต็อก และคงเหลือ ≤ ขั้นต่ำ)
type ListFilter struct {
	Query      string
	CategoryID string
	Kind       string
	OnlyLow    bool
	Limit      int
	Offset     int
}

// Validate คืน error code ("" = ผ่าน)
func (f ListFilter) Validate() string {
	if f.Kind != "" && f.Kind != KindForSale && f.Kind != KindMaterial {
		return "product_kind_invalid"
	}
	if f.CategoryID != "" && !utils.IsUUID(f.CategoryID) {
		return "product_category_invalid"
	}
	return ""
}

// likePattern คำค้นสำหรับ ILIKE "มีคำนี้อยู่ตรงไหนก็ได้" — ตัวอักษรพิเศษของ LIKE (\ % _) ให้ค้นเป็นตัวอักษรธรรมดา
// เช่น "50%" → "%50\%%" (ไม่ใช่ "50 อะไรก็ได้")
func likePattern(query string) string {
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.TrimSpace(query))
	return "%" + escaped + "%"
}

// Counts จำนวนสินค้าทั้งร้าน ไว้ทำตัวเลขบนแท็บ (ทั้งหมด / สินค้าขาย / วัสดุ) และปุ่มเฉพาะใกล้หมด
type Counts struct {
	All      int `json:"all"`
	ForSale  int `json:"for_sale"`
	Material int `json:"material"`
	Low      int `json:"low"`
}
