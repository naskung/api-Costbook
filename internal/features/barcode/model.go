package barcode

// Product ข้อมูลสินค้าที่หาเจอจากเลขบาร์โค้ด (ไว้กรอกฟอร์มสินค้าให้ ไม่ได้บันทึกลง DB)
// name = ชื่อสินค้า (มีชื่อไทยใช้ชื่อไทยก่อน), brand = ยี่ห้อ, quantity = ขนาด เช่น "600 mL" (ไม่มี = "")
type Product struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Brand    string `json:"brand"`
	Quantity string `json:"quantity"`
}
