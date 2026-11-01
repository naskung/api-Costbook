package ai

import (
	"strings"
	"time"
	"unicode/utf8"
)

// ขีดจำกัดของแชท (กัน request ใหญ่เกิน / ค่า AI บานปลาย)
const (
	maxChatMessages  = 40
	maxMessageLength = 4000
)

// MarketCheck ผลเช็คราคาตลาด 1 ครั้งของสินค้า 1 ตัว (ที่ API ตอบกลับ)
// our_* = ต้นทุน / ราคาขายของร้าน ณ วันที่เช็ค, ราคาทุกช่องเป็นบาทต่อหน่วยของร้าน, 0 = ไม่รู้
type MarketCheck struct {
	ID           string  `json:"id"`
	ProductID    string  `json:"product_id"`
	OurCostPrice float64 `json:"our_cost_price"`
	OurSellPrice float64 `json:"our_sell_price"`
	MarketReport
	CheckedAt time.Time `json:"checked_at"`
}

// MarketReport ข้อมูลที่ Claude ส่งกลับมาผ่านเครื่องมือ save_market_report
// retail = ขายปลีก, wholesale = ขายส่ง, market_cost = ต้นทุนต่อหน่วยที่ร้านอื่นน่าจะใช้
// confidence = low / medium / high (ข้อมูลที่เจอมากแค่ไหน)
type MarketReport struct {
	Summary        string           `json:"summary"`
	RetailLow      float64          `json:"retail_low"`
	RetailHigh     float64          `json:"retail_high"`
	RetailTypical  float64          `json:"retail_typical"`
	WholesaleLow   float64          `json:"wholesale_low"`
	WholesaleHigh  float64          `json:"wholesale_high"`
	WholesaleNote  string           `json:"wholesale_note"`
	MarketCost     float64          `json:"market_cost"`
	CostItems      []MarketCostItem `json:"cost_items"`
	SuggestedPrice float64          `json:"suggested_price"`
	Advice         string           `json:"advice"`
	Confidence     string           `json:"confidence"`
	Sources        []MarketSource   `json:"sources"`
	// SearchSuggestions แถบคำค้นแนะนำของ Google (HTML) — มีเมื่อเช็คด้วย Gemini (ค้นผ่าน Google)
	// กติกาของ Google: ต้องแสดงคู่กับผลเสมอ ห้ามแก้ไข / Claude ไม่มี = ""
	SearchSuggestions string `json:"search_suggestions"`
}

// MarketCostItem ต้นทุน 1 รายการที่คนทำขายน่าจะต้องจ่าย (ต่อหน่วย)
type MarketCostItem struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
	Note   string  `json:"note"`
}

// MarketSource แหล่งที่มาของราคา kind = retail / wholesale / material
type MarketSource struct {
	Title string  `json:"title"`
	URL   string  `json:"url"`
	Price float64 `json:"price"`
	Kind  string  `json:"kind"`
}

// ChatMessage ข้อความ 1 อันในแชท role = user / assistant
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest ประวัติแชทที่ Flutter ส่งมาทั้งหมด (API ไม่ได้เก็บแชทไว้)
// ต้องเริ่มด้วย user สลับกับ assistant และจบด้วย user (ข้อความล่าสุดที่ถาม)
type ChatRequest struct {
	Messages []ChatMessage `json:"messages"`
}

// Validate ตรวจประวัติแชท ถ้าผิดคืน error code
func (req ChatRequest) Validate() string {
	count := len(req.Messages)
	if count == 0 || count > maxChatMessages || count%2 == 0 {
		return "ai_messages_invalid"
	}
	for i, message := range req.Messages {
		wantRole := "user"
		if i%2 == 1 {
			wantRole = "assistant"
		}
		if message.Role != wantRole || strings.TrimSpace(message.Content) == "" {
			return "ai_messages_invalid"
		}
		if utf8.RuneCountInString(message.Content) > maxMessageLength {
			return "ai_message_too_long"
		}
	}
	return ""
}

// ChatReply คำตอบของ AI + งานที่ AI เสนอให้ทำ (ผู้ใช้ต้องกดยืนยันเอง)
type ChatReply struct {
	Reply   string   `json:"reply"`
	Actions []Action `json:"actions"`
}

// Action งานที่ AI เสนอ 1 งาน — API ไม่ได้ทำให้ Flutter เรียก API เส้นเดิมเมื่อผู้ใช้กดยืนยัน
// type: create_product (data = ข้อมูลสินค้าใหม่ → POST /products)
//
//	update_product (data = ข้อมูลสินค้าทั้งตัวหลังแก้ → PUT /products/{product_id}, changes = ช่องที่เปลี่ยน)
//	stock_in / stock_out (data → POST /stock/in หรือ /stock/out)
//	add_transaction (data → POST /transactions)
//	market_check (→ POST /products/{product_id}/market)
type Action struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	Reason      string   `json:"reason"`
	ProductID   string   `json:"product_id"`
	ProductName string   `json:"product_name"`
	Changes     []Change `json:"changes"`
	Data        any      `json:"data"`
}

// Change ช่องที่ AI เสนอให้แก้ field = ชื่อช่องใน JSON ของสินค้า เช่น sell_price
type Change struct {
	Field string `json:"field"`
	From  any    `json:"from"`
	To    any    `json:"to"`
}
