package ai

import (
	"strings"
	"testing"
)

func TestSearchPlan(t *testing.T) {
	plan := searchPlan([]plannedQuery{
		{Text: "พวงกุญแจ 3D", Source: searchMarketplace},
		{Text: "ราคาหมูเนื้อแดง", Source: searchCentralMarket},
	})
	if len(plan) != 3 {
		t.Fatalf("ร้านออนไลน์ 2 ครั้ง (Shopee + Lazada) + ตลาดกลาง 1 ครั้ง ได้ %d ครั้ง", len(plan))
	}
	if plan[0].label != "Shopee" || plan[0].query.Sites[0] != "shopee.co.th" || plan[1].label != "Lazada" || plan[1].query.Sites[0] != "lazada.co.th" {
		t.Errorf("ร้านออนไลน์ต้องค้นแยก Shopee / Lazada ได้ %+v", plan[:2])
	}
	central := plan[2]
	if central.label != centralMarketLabel || !central.query.PreferSites || !central.query.Recent || !strings.Contains(strings.Join(central.query.Sites, ","), "talaadthai.com") {
		t.Errorf("ของสดต้องค้นตลาดกลาง เน้นเว็บทางการ ข้อมูลล่าสุด ได้ %+v", central)
	}
}

func TestSearchPlan_Limit(t *testing.T) {
	queries := []plannedQuery{}
	for _, text := range []string{"ก", "ข", "ค", "ง"} {
		queries = append(queries, plannedQuery{Text: text, Source: searchMarketplace})
	}
	if plan := searchPlan(queries); len(plan) != maxSearches {
		t.Errorf("ค้นไม่เกิน %d ครั้งต่อสินค้า (= เครดิต) ได้ %d", maxSearches, len(plan))
	}
}

func TestCleanQueries(t *testing.T) {
	got := cleanQueries([]plannedQuery{
		{Text: " พวงกุญแจ ", Source: searchMarketplace},
		{Text: "พวงกุญแจ", Source: searchMarketplace},
		{Text: "พวงกุญแจ", Source: searchCentralMarket},
		{Text: "  ", Source: searchMarketplace},
		{Text: "เส้น PLA", Source: "อะไรก็ไม่รู้"},
	})
	want := []plannedQuery{
		{Text: "พวงกุญแจ", Source: searchMarketplace},
		{Text: "พวงกุญแจ", Source: searchCentralMarket},
		{Text: "เส้น PLA", Source: searchMarketplace},
	}
	if len(got) != len(want) {
		t.Fatalf("ได้ %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ตัดช่องว่าง / ไม่ว่าง / ไม่ซ้ำ (คำเดียวกันคนละแหล่งได้) / แหล่งที่ไม่รู้จัก = ร้านออนไลน์ ได้ %+v", got)
		}
	}
}
