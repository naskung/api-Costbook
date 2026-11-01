package ai

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/naskung/api-Costbook/internal/features/stock"
	"github.com/naskung/api-Costbook/internal/features/transaction"
)

func testToolbox() toolbox {
	return toolbox{shop: fakeShop{}, repo: &fakeRepository{}, today: testToday, canSearch: true}
}

// runTool เรียกเครื่องมือด้วย input แบบ JSON
func runTool(t *testing.T, name, input string) (string, *Action, error) {
	t.Helper()
	return testToolbox().run(context.Background(), name, []byte(input))
}

// wantToolError ต้องได้ error ที่ส่งกลับให้ AI แก้ และข้อความมีคำว่า want
func wantToolError(t *testing.T, err error, want string) {
	t.Helper()
	var toolErr toolError
	if !errors.As(err, &toolErr) || !strings.Contains(err.Error(), want) {
		t.Errorf("ต้องได้ toolError ที่มีคำว่า %q ได้ %v", want, err)
	}
}

func TestToolbox_UpdateProduct(t *testing.T) {
	_, action, err := runTool(t, "propose_update_product", `{"product_id":"`+caseID+`","sell_price":290,"unit":"ชิ้น","reason":"ตลาดขาย 259–390"}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(action.Changes) != 1 || action.Changes[0] != (Change{Field: "sell_price", From: 250.0, To: 290.0}) {
		t.Errorf("หน่วยเหมือนเดิมไม่นับว่าเปลี่ยน ต้องเปลี่ยนแค่ราคา ได้ %+v", action.Changes)
	}
	req := action.Data.(productProposal).ProductRequest
	if req.SellPrice != 290 || req.Name != "เคสโทรศัพท์" || req.CostPrice != 52.8 || !*req.TrackStock {
		t.Errorf("data ต้องเป็นสินค้าทั้งตัวหลังแก้ ได้ %+v", req)
	}
}

func TestToolbox_UpdateProduct_NoChange(t *testing.T) {
	_, _, err := runTool(t, "propose_update_product", `{"product_id":"`+caseID+`","sell_price":250,"reason":"x"}`)
	wantToolError(t, err, "ไม่มีช่องไหนเปลี่ยน")
}

func TestToolbox_CreateProduct(t *testing.T) {
	_, action, err := runTool(t, "propose_create_product", `{"name":"พวงกุญแจ","sell_price":89,"reason":"x"}`)
	if err != nil {
		t.Fatal(err)
	}
	req := action.Data.(productProposal).ProductRequest
	if action.Type != actionCreateProduct || !*req.TrackStock || req.CustomFields == nil {
		t.Errorf("ไม่ส่ง track_stock = นับสต็อก, custom_fields = {} ได้ %+v", req)
	}

	_, _, err = runTool(t, "propose_create_product", `{"name":"","sell_price":89,"reason":"x"}`)
	wantToolError(t, err, "product_name_required")
}

func TestToolbox_StockOut_NotEnough(t *testing.T) {
	_, _, err := runTool(t, "propose_stock_out", `{"product_id":"`+caseID+`","quantity":9,"reason":"ของเสีย"}`)
	wantToolError(t, err, "คงเหลือแค่ 8 ชิ้น")

	_, action, err := runTool(t, "propose_stock_out", `{"product_id":"`+caseID+`","quantity":8,"unit_cost":99,"reason":"ของเสีย"}`)
	if err != nil {
		t.Fatal(err)
	}
	if req := action.Data.(stock.MovementRequest); req.UnitCost != 0 {
		t.Errorf("เบิกออกไม่ใช้ต้นทุน ได้ %+v", req)
	}
}

func TestToolbox_Transaction(t *testing.T) {
	_, action, err := runTool(t, "propose_transaction", `{"type":"expense","category":"rent","amount":5000,"reason":"ค่าเช่า"}`)
	if err != nil {
		t.Fatal(err)
	}
	if req := action.Data.(transaction.TransactionRequest); req.Date != "2026-10-31" {
		t.Errorf("ไม่ใส่วันที่ = วันนี้ ได้ %+v", req)
	}

	_, _, err = runTool(t, "propose_transaction", `{"type":"income","category":"rent","amount":5000,"reason":"x"}`)
	wantToolError(t, err, "ใช้กับ income ไม่ได้")

	_, _, err = runTool(t, "propose_transaction", `{"type":"expense","category":"rent","amount":0,"reason":"x"}`)
	wantToolError(t, err, "transaction_amount_invalid")
}

func TestToolbox_Dashboard_Month(t *testing.T) {
	text, _, err := runTool(t, "get_dashboard", `{}`)
	if err != nil || !strings.Contains(text, `"month":"2026-10"`) {
		t.Errorf("ไม่ใส่เดือน = เดือนนี้ ได้ %s %v", text, err)
	}
	_, _, err = runTool(t, "get_dashboard", `{"month":"ตุลา"}`)
	wantToolError(t, err, "YYYY-MM")
}

func TestToolbox_ListProducts(t *testing.T) {
	text, _, err := runTool(t, "list_products", `{}`)
	var items []map[string]any
	json.Unmarshal([]byte(text), &items)
	if err != nil || len(items) != 1 || items[0]["id"] != caseID || items[0]["stock"] != 8.0 {
		t.Errorf("ได้ %s %v", text, err)
	}
}

func TestToolbox_MarketCheck_NotChecked(t *testing.T) {
	text, _, err := runTool(t, "get_market_check", `{"product_id":"`+caseID+`"}`)
	if err != nil || !strings.Contains(text, "ยังไม่เคยเช็ค") {
		t.Errorf("ยังไม่เคยเช็ค ต้องบอก AI ให้เสนอเช็คใหม่ ได้ %q %v", text, err)
	}
}

func TestToolbox_UnknownTool(t *testing.T) {
	_, _, err := runTool(t, "delete_everything", `{}`)
	wantToolError(t, err, "ไม่มีเครื่องมือ")
}

func TestToolbox_MarketCheck_NeedsWebSearch(t *testing.T) {
	_, action, err := runTool(t, "propose_market_check", `{"product_id":"`+caseID+`","reason":"x"}`)
	if err != nil || action.Type != actionMarketCheck {
		t.Errorf("AI ที่ค้นเว็บได้ เสนอเช็คราคาตลาดได้ ได้ %+v %v", action, err)
	}
	free := toolbox{shop: fakeShop{}, repo: &fakeRepository{}, today: testToday}
	_, _, err = free.run(context.Background(), "propose_market_check", []byte(`{"product_id":"`+caseID+`","reason":"x"}`))
	wantToolError(t, err, "ค้นเว็บไม่ได้")
}

func TestToolbox_CreateProduct_Category(t *testing.T) {
	_, action, err := runTool(t, "propose_create_product", `{"name":"พวงกุญแจ","category":" งานพิมพ์ 3d ","sell_price":89,"reason":"x"}`)
	if err != nil {
		t.Fatal(err)
	}
	req := action.Data.(productProposal).ProductRequest
	if req.CategoryID == nil || *req.CategoryID != printCategoryID {
		t.Errorf("category_id = %v, ต้องได้ id ของหมวด งานพิมพ์ 3D (ชื่อไม่สนตัวพิมพ์ / ช่องว่าง)", req.CategoryID)
	}
	if name := action.Data.(productProposal).Category; name != "งานพิมพ์ 3d" {
		t.Errorf("ชื่อหมวดไว้โชว์ = %q, ต้องได้ชื่อที่ AI ส่งมา (ตัดช่องว่าง)", name)
	}

	_, _, err = runTool(t, "propose_create_product", `{"name":"พวงกุญแจ","category":"ของเล่น","sell_price":89,"reason":"x"}`)
	wantToolError(t, err, "ยังไม่มีหมวด")
}

func TestToolbox_UpdateProduct_Category(t *testing.T) {
	_, action, err := runTool(t, "propose_update_product", `{"product_id":"`+caseID+`","category":"งานพิมพ์ 3D","reason":"จัดหมวด"}`)
	if err != nil {
		t.Fatal(err)
	}
	req := action.Data.(productProposal).ProductRequest
	if req.CategoryID == nil || *req.CategoryID != printCategoryID {
		t.Errorf("category_id = %v, ต้องได้ id ของหมวด งานพิมพ์ 3D", req.CategoryID)
	}
	if len(action.Changes) != 1 || action.Changes[0].Field != "category" || action.Changes[0].To != "งานพิมพ์ 3D" {
		t.Errorf("changes = %+v, ต้องบอกว่าเปลี่ยนหมวดเป็น งานพิมพ์ 3D (แสดงชื่อ ไม่ใช่ id)", action.Changes)
	}
}

func TestToolbox_ShopInfo_Categories(t *testing.T) {
	text, _, err := runTool(t, "get_shop_info", `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "งานพิมพ์ 3D") {
		t.Errorf("ข้อมูลร้าน = %s, ต้องมีรายชื่อหมวดให้ AI เลือก", text)
	}
}
