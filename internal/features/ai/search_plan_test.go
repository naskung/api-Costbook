package ai

import (
	"strings"
	"testing"
)

func TestSearchPlan(t *testing.T) {
	plan := searchPlan([]plannedQuery{
		{Text: "พวงกุญแจ 3D", Source: searchMarketplace},
		{Text: "ราคาหมูเนื้อแดง", Source: searchCentralMarket},
		{Text: "เส้น PLA", Source: searchMarketplace},
	})
	if len(plan) != 5 {
		t.Fatalf("คำแรก 3 ครั้ง (ทั้งเว็บไทย + Shopee + Lazada) + ตลาดกลาง 1 + คำถัดไปทั้งเว็บไทย 1 ได้ %d ครั้ง", len(plan))
	}
	thaiWeb := plan[0].query
	if plan[0].label != thaiShopsLabel || len(thaiWeb.Sites) != 0 || strings.Join(thaiWeb.ExcludeSites, ",") != "shopee.co.th,lazada.co.th" {
		t.Errorf("คำแรกต้องค้นทั้งเว็บไทย ไม่เอา Shopee / Lazada (ค้นแยกแล้ว) ได้ %+v", plan[0])
	}
	if plan[1].label != "Shopee" || plan[1].query.Sites[0] != "shopee.co.th" || plan[2].label != "Lazada" || plan[2].query.Sites[0] != "lazada.co.th" {
		t.Errorf("คำแรกต้องค้นแยก Shopee / Lazada ด้วย ได้ %+v", plan[1:3])
	}
	last := plan[4]
	if last.label != thaiShopsLabel || last.query.Text != "เส้น PLA" || len(last.query.Sites) != 0 || len(last.query.ExcludeSites) != 0 {
		t.Errorf("คำถัดไปค้นทั้งเว็บไทยครั้งเดียว (รวม Shopee / Lazada) ได้ %+v", last)
	}
	central := plan[3]
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
