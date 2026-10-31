package ai

import (
	"strings"
	"testing"

	"github.com/naskung/api-Costbook/internal/features/company"
	"github.com/naskung/api-Costbook/internal/features/cost"
	"github.com/naskung/api-Costbook/internal/features/field"
	"github.com/naskung/api-Costbook/internal/features/product"
)

var testFields = []field.ProductField{{ID: "f1", Name: "วัสดุ"}, {ID: "f2", Name: "สี"}, {ID: "f3", Name: "ขนาด"}}

func TestFieldValues(t *testing.T) {
	p := product.Product{CustomFields: map[string]string{"f2": "ดำ", "f1": "PLA", "f3": " "}}
	got := strings.Join(fieldValues(p, testFields), ", ")
	if got != "วัสดุ: PLA, สี: ดำ" {
		t.Errorf("ได้ %q ต้องเรียงตามลำดับช่อง และไม่ใส่ช่องที่ว่าง", got)
	}
}

func TestResearchPrompt(t *testing.T) {
	p := product.Product{
		Name: "เคสโทรศัพท์พิมพ์ 3D", Category: "งานพิมพ์ 3D", Unit: "ชิ้น", CostPrice: 52.8, SellPrice: 250,
		CustomFields: map[string]string{"f1": "PLA"},
	}
	sheet := cost.CostSheet{BatchQuantity: 4, Items: []cost.CostItem{
		{Kind: "fixed", Name: "เส้น PLA", Quantity: 160, Unit: "กรัม", UnitPrice: 0.45, Amount: 72},
		{Kind: "percent", Name: "เผื่อพิมพ์เสีย", Percent: 10, Amount: 19.2},
	}}
	got := researchPrompt(p, sheet, testFields, company.Company{VATRegistered: true, PricesIncludeVAT: true})

	for _, want := range []string{
		"ชื่อ: เคสโทรศัพท์พิมพ์ 3D", "หมวด: งานพิมพ์ 3D", "วัสดุ: PLA",
		"ต้นทุนของร้าน: 52.80 บาท/ชิ้น", "ราคาขายของร้าน: 250.00 บาท/ชิ้น (รวม VAT แล้ว",
		"ได้ 4 ชิ้น", "เส้น PLA 160 กรัม × 0.45: 72.00 บาท", "เผื่อพิมพ์เสีย 10% ของต้นทุนอื่น: 19.20 บาท",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("ข้อความต้องมี %q\nได้:\n%s", want, got)
		}
	}
	if strings.Contains(got, "รายละเอียด") {
		t.Error("ไม่ได้กรอกรายละเอียด ไม่ต้องใส่บรรทัดรายละเอียด")
	}
}

func TestResearchPrompt_NoCostSheet(t *testing.T) {
	got := researchPrompt(product.Product{Name: "ชาไทย"}, cost.CostSheet{}, nil, company.Company{VATRegistered: false})
	if !strings.Contains(got, "หน่วยขาย: ชิ้น") || !strings.Contains(got, "ร้านไม่จด VAT") || strings.Contains(got, "รายการต้นทุน") {
		t.Errorf("ไม่มีหน่วย = ชิ้น, ร้านไม่จด VAT, ไม่มีใบต้นทุน ไม่ต้องใส่รายการต้นทุน\nได้:\n%s", got)
	}
}

func TestVatNote(t *testing.T) {
	if got := vatNote(company.Company{VATRegistered: true}); !strings.Contains(got, "ยังไม่รวม VAT") {
		t.Errorf("ร้านจด VAT ราคาไม่รวม VAT ได้ %q", got)
	}
}

func TestCleanReport(t *testing.T) {
	got := cleanReport(MarketReport{
		RetailLow: 390, RetailHigh: 199.555, RetailTypical: 250, WholesaleLow: -5, MarketCost: 45.678,
		Summary: "  ราคาส่วนใหญ่ 199–390 บาท  ", Confidence: "very high",
		CostItems: []MarketCostItem{{Name: "เส้น PLA", Amount: 18}, {Name: " ", Amount: 1}},
		Sources: []MarketSource{
			{Title: "Shopee", URL: "https://shopee.co.th/x", Price: 259, Kind: "retail"},
			{Title: "แปลก", URL: "javascript:alert(1)", Price: 1},
			{Title: "โรงงาน", URL: "http://factory.example", Price: 120, Kind: "unknown"},
		},
	})

	if got.RetailLow != 199.56 || got.RetailHigh != 390 {
		t.Errorf("ต่ำสุด > สูงสุด ต้องสลับ + ปัดเศษ ได้ %v–%v", got.RetailLow, got.RetailHigh)
	}
	if got.WholesaleLow != 0 || got.MarketCost != 45.68 {
		t.Errorf("ติดลบ → 0, ปัดเศษ 2 ตำแหน่ง ได้ %v, %v", got.WholesaleLow, got.MarketCost)
	}
	if got.Summary != "ราคาส่วนใหญ่ 199–390 บาท" || got.Confidence != confidenceLow {
		t.Errorf("ตัดช่องว่าง + ความมั่นใจแปลก ๆ → low ได้ %q, %q", got.Summary, got.Confidence)
	}
	if len(got.CostItems) != 1 {
		t.Errorf("รายการต้นทุนที่ไม่มีชื่อต้องตัดทิ้ง ได้ %+v", got.CostItems)
	}
	if len(got.Sources) != 2 || got.Sources[1].Kind != sourceRetail {
		t.Errorf("ตัดลิงก์ที่ไม่ใช่ http(s) + ชนิดแปลก ๆ → retail ได้ %+v", got.Sources)
	}
}

func TestCleanReport_LimitsLists(t *testing.T) {
	sources := []MarketSource{}
	for i := 0; i < 15; i++ {
		sources = append(sources, MarketSource{URL: "https://example.com", Kind: sourceRetail})
	}
	got := cleanReport(MarketReport{Sources: sources, Advice: strings.Repeat("ก", 2500)})
	if len(got.Sources) != maxReportItems {
		t.Errorf("แหล่งที่มาต้องไม่เกิน %d ได้ %d", maxReportItems, len(got.Sources))
	}
	if len([]rune(got.Advice)) != maxReportText {
		t.Errorf("คำแนะนำต้องไม่เกิน %d ตัวอักษร ได้ %d", maxReportText, len([]rune(got.Advice)))
	}
	if got.CostItems == nil || len(got.CostItems) != 0 {
		t.Error("ไม่มีรายการต้นทุน ต้องได้ list ว่าง (JSON = []) ไม่ใช่ nil")
	}
}
