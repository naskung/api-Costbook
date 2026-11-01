package product

import (
	"strings"
	"testing"
)

func TestProductRequest_Validate(t *testing.T) {
	tooManyFields := map[string]string{}
	for i := 0; i <= maxCustomFields; i++ {
		tooManyFields[strings.Repeat("x", i+1)] = "ค่า"
	}

	categoryID := "c1111111-1111-1111-1111-111111111111"
	noCategory := ""
	badCategory := "งานพิมพ์ 3D"

	tests := []struct {
		name string
		req  ProductRequest
		want string
	}{
		{"มีหมวด (id) ผ่าน", ProductRequest{Name: "เคส", CategoryID: &categoryID}, ""},
		{"ส่งหมวดว่าง = ไม่มีหมวด ผ่าน", ProductRequest{Name: "เคส", CategoryID: &noCategory}, ""},
		{"ส่งชื่อหมวดแทน id ไม่ผ่าน", ProductRequest{Name: "เคส", CategoryID: &badCategory}, "product_category_invalid"},
		{"ข้อมูลครบ ผ่าน", ProductRequest{Name: "กาแฟ", Unit: "แก้ว", CostPrice: 20, SellPrice: 45, MinStock: 5}, ""},
		{"ราคาเป็น 0 ผ่าน", ProductRequest{Name: "กาแฟ"}, ""},
		{"มีช่องเพิ่มเติม ผ่าน", ProductRequest{Name: "เคส", CustomFields: map[string]string{"f1": "PLA"}}, ""},
		{"ชื่อว่าง ไม่ผ่าน", ProductRequest{Name: "", SellPrice: 45}, "product_name_required"},
		{"ชื่อมีแต่ช่องว่าง ไม่ผ่าน", ProductRequest{Name: "   ", SellPrice: 45}, "product_name_required"},
		{"ต้นทุนติดลบ ไม่ผ่าน", ProductRequest{Name: "กาแฟ", CostPrice: -1}, "product_price_invalid"},
		{"ราคาขายติดลบ ไม่ผ่าน", ProductRequest{Name: "กาแฟ", SellPrice: -0.5}, "product_price_invalid"},
		{"จำนวนขั้นต่ำติดลบ ไม่ผ่าน", ProductRequest{Name: "กาแฟ", MinStock: -1}, "product_min_stock_invalid"},
		{"ช่องเพิ่มเติมเยอะเกิน ไม่ผ่าน", ProductRequest{Name: "เคส", CustomFields: tooManyFields}, "product_custom_fields_invalid"},
		{"ค่าในช่องยาวเกิน ไม่ผ่าน", ProductRequest{Name: "เคส", CustomFields: map[string]string{"f1": strings.Repeat("ก", 200)}}, "product_custom_fields_invalid"},
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

func TestProductRequest_TrackStock(t *testing.T) {
	off := false
	on := true

	tests := []struct {
		name string
		req  ProductRequest
		want bool
	}{
		{"ไม่ส่งมา ได้นับสต็อก", ProductRequest{}, true},
		{"ส่ง true มา ได้นับสต็อก", ProductRequest{TrackStock: &on}, true},
		{"ส่ง false มา ได้ไม่นับสต็อก", ProductRequest{TrackStock: &off}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.req.trackStock(); got != tt.want {
				t.Errorf("trackStock() = %v, ต้องได้ %v", got, tt.want)
			}
		})
	}
}

func TestProductRequest_CustomFields(t *testing.T) {
	if got := (ProductRequest{}).customFields(); got == nil || len(got) != 0 {
		t.Errorf("ไม่ส่งมา ได้ %#v, ต้องได้ map ว่าง (ไม่ใช่ nil)", got)
	}
}

func TestProductRequest_ForSale(t *testing.T) {
	no := false
	if !(ProductRequest{}).forSale() {
		t.Error("ไม่ส่ง for_sale มา ต้องเป็นสินค้าที่ขาย")
	}
	if (ProductRequest{ForSale: &no}).forSale() {
		t.Error("ส่ง for_sale = false ต้องเป็นวัสดุ (ไม่ขาย)")
	}
}

func TestListFilter_Validate(t *testing.T) {
	tests := []struct {
		name   string
		filter ListFilter
		want   string
	}{
		{"ไม่กรองอะไร ผ่าน", ListFilter{}, ""},
		{"สินค้าขาย / วัสดุ ผ่าน", ListFilter{Kind: KindMaterial}, ""},
		{"kind ไม่มีจริง", ListFilter{Kind: "food"}, "product_kind_invalid"},
		{"หมวดไม่ใช่ UUID", ListFilter{CategoryID: "drink"}, "product_category_invalid"},
	}
	for _, tt := range tests {
		if got := tt.filter.Validate(); got != tt.want {
			t.Errorf("%s: ได้ %q ต้องได้ %q", tt.name, got, tt.want)
		}
	}
}

func TestLikePattern(t *testing.T) {
	tests := map[string]string{
		" กาแฟ ":  "%กาแฟ%",
		"50%":     `%50\%%`,
		"a_b":     `%a\_b%`,
		`c:\temp`: `%c:\\temp%`,
	}
	for query, want := range tests {
		if got := likePattern(query); got != want {
			t.Errorf("likePattern(%q) = %q ต้องได้ %q", query, got, want)
		}
	}
}
