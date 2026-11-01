package ai

import "strings"

// แหล่งที่ค้นราคา (AI เลือกให้แต่ละคำค้น)
// marketplace = ร้านออนไลน์ Shopee + Lazada (ของทั่วไป ของสำเร็จรูป งานทำมือ วัตถุดิบที่ไม่ใช่ของสด)
// central_market = ตลาดกลาง / ราคาทางการ (ของสด: เนื้อสัตว์ ผัก ผลไม้ อาหารทะเล ไข่ ข้าว)
const (
	searchMarketplace   = "marketplace"
	searchCentralMarket = "central_market"
)

// maxSearches ค้นไม่เกินกี่ครั้งต่อการเช็ค 1 สินค้า (= เครดิต Tavily) → ฟรี 1,000 / เดือน ≈ 170 สินค้า
const maxSearches = 6

// marketplaceSites ร้านออนไลน์ — ค้นแยกทีละเว็บ (ค้นรวมกัน ผลเป็น Shopee หมด Lazada ไม่ติด)
var marketplaceSites = []struct {
	domain string
	label  string
}{
	{"shopee.co.th", "Shopee"},
	{"lazada.co.th", "Lazada"},
}

// centralMarketSites ตลาดกลาง / หน่วยงานที่ประกาศราคาของสด (ค้นแบบเน้นเว็บเหล่านี้ เว็บราคาเกษตรอื่นก็ได้)
var centralMarketSites = []string{"talaadthai.com", "simummuangmarket.com", "dit.go.th", "moc.go.th", "oae.go.th"}

const centralMarketLabel = "ตลาดกลาง / ราคาทางการ"

// plannedQuery คำค้นที่ AI คิด + แหล่งที่ควรค้น
type plannedQuery struct {
	Text   string `json:"text"`
	Source string `json:"source"`
}

// plannedSearch การค้นจริง 1 ครั้ง + ป้ายบอกแหล่ง (ใส่ให้ผลค้นหา ให้ AI รู้ว่าราคามาจากไหน)
type plannedSearch struct {
	query SearchQuery
	label string
}

// searchPlan แปลงคำค้นที่ AI คิด เป็นการค้นจริง ไม่เกิน maxSearches ครั้ง
// ร้านออนไลน์ → ค้น Shopee 1 ครั้ง + Lazada 1 ครั้ง / ตลาดกลาง → ค้น 1 ครั้ง เน้นเว็บตลาดกลาง เฉพาะข้อมูล 1 เดือนล่าสุด
func searchPlan(queries []plannedQuery) []plannedSearch {
	plan := []plannedSearch{}
	for _, q := range queries {
		if q.Source == searchCentralMarket {
			query := SearchQuery{Text: q.Text, Sites: centralMarketSites, PreferSites: true, Recent: true}
			plan = append(plan, plannedSearch{query: query, label: centralMarketLabel})
			continue
		}
		for _, site := range marketplaceSites {
			plan = append(plan, plannedSearch{query: SearchQuery{Text: q.Text, Sites: []string{site.domain}}, label: site.label})
		}
	}
	if len(plan) > maxSearches {
		plan = plan[:maxSearches]
	}
	return plan
}

// cleanQueries คำค้นที่ใช้ได้: ตัดช่องว่าง ไม่ว่าง ไม่ซ้ำ (คำเดียวกันคนละแหล่งได้) แหล่งที่ไม่รู้จัก = ร้านออนไลน์
func cleanQueries(queries []plannedQuery) []plannedQuery {
	cleaned := []plannedQuery{}
	seen := map[string]bool{}
	for _, q := range queries {
		q.Text = strings.TrimSpace(q.Text)
		if q.Source != searchCentralMarket {
			q.Source = searchMarketplace
		}
		key := q.Source + "|" + q.Text
		if q.Text == "" || seen[key] {
			continue
		}
		seen[key] = true
		cleaned = append(cleaned, q)
	}
	return cleaned
}
