package product

import "strings"

// ความยาวสูงสุดของช่องเพิ่มเติม กันข้อมูลใหญ่ผิดปกติ
const (
	maxCustomFields     = 50
	maxCustomFieldValue = 500
)

// Product ข้อมูลสินค้า (ชื่อ json ต้องตรงกับ ProductModel ฝั่ง Flutter)
// stock เปลี่ยนผ่านหน้าสต็อก / เอกสารขายเท่านั้น, batch_quantity แก้ในหน้าคำนวณต้นทุน
// track_stock = false → งานสั่งทำ / ค่าบริการ ไม่นับสต็อก ขายได้ไม่จำกัด
// custom_fields = ค่าของช่องเพิ่มเติมที่ร้านตั้งเอง {"<id ช่อง>": "ค่า"} (ดู /product-fields)
// image_version = เวลาที่เปลี่ยนรูปล่าสุด (0 = ไม่มีรูป) → รูปอยู่ที่ GET /products/{id}/image?v=<image_version>
type Product struct {
	ID            string            `json:"id"`
	Code          string            `json:"code"`
	Name          string            `json:"name"`
	Category      string            `json:"category"`
	Description   string            `json:"description"`
	Unit          string            `json:"unit"`
	CostPrice     float64           `json:"cost_price"`
	SellPrice     float64           `json:"sell_price"`
	Stock         int               `json:"stock"`
	MinStock      int               `json:"min_stock"`
	BatchQuantity int               `json:"batch_quantity"`
	TrackStock    bool              `json:"track_stock"`
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
// ไม่ส่ง track_stock / custom_fields มา: ตอนเพิ่ม = นับสต็อก / ไม่มีช่องเพิ่มเติม, ตอนแก้ = ใช้ค่าเดิม
type ProductRequest struct {
	Code         string            `json:"code"`
	Name         string            `json:"name"`
	Category     string            `json:"category"`
	Description  string            `json:"description"`
	Unit         string            `json:"unit"`
	CostPrice    float64           `json:"cost_price"`
	SellPrice    float64           `json:"sell_price"`
	MinStock     int               `json:"min_stock"`
	TrackStock   *bool             `json:"track_stock"`
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
