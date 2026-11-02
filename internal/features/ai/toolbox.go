package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/naskung/api-Costbook/internal/features/product"
	"github.com/naskung/api-Costbook/internal/features/stock"
	"github.com/naskung/api-Costbook/internal/features/transaction"
	"github.com/naskung/api-Costbook/internal/utils"
)

// proposedText ข้อความที่ตอบ AI เมื่อเสนองานสำเร็จ (งานยังไม่ได้ทำ รอผู้ใช้กดยืนยัน)
const proposedText = "ส่งข้อเสนอให้ผู้ใช้แล้ว ยังไม่ได้ทำ ผู้ใช้ต้องกดยืนยันในหน้าแชทเอง"

// toolError ข้อผิดพลาดที่ส่งกลับให้ AI แก้เอง (เช่น id สินค้าผิด) — ไม่ใช่ error ของระบบ
type toolError string

func (e toolError) Error() string { return string(e) }

// toolbox ทำงานตามเครื่องมือที่ AI เรียกในแชท
// canSearch = AI ที่ใช้อยู่ค้นเว็บได้ไหม (Claude ได้เสมอ / Gemini ได้เมื่อมี TAVILY_API_KEY หรือเปิด GEMINI_WEB_SEARCH, ไม่ได้ → เสนอเช็คราคาตลาดไม่ได้)
type toolbox struct {
	shop      ShopReader
	repo      Repository
	today     time.Time
	canSearch bool
}

// run ทำงานของเครื่องมือ 1 ครั้ง → ข้อความผลลัพธ์ (JSON) + งานที่เสนอ (ถ้ามี)
// error แบบ toolError = ส่งกลับให้ AI แก้, error อื่น = ระบบพัง (หยุดแชท)
func (t toolbox) run(ctx context.Context, name string, input []byte) (string, *Action, error) {
	switch name {
	case "get_shop_info":
		return t.shopInfo(ctx)
	case "list_products":
		return t.listProducts(ctx)
	case "get_cost_sheet":
		return t.costSheet(ctx, input)
	case "get_market_check":
		return t.marketCheck(ctx, input)
	case "get_dashboard":
		return t.dashboard(ctx, input)
	case "list_transactions":
		return t.transactions(ctx, input)
	case "propose_create_product":
		return t.proposeCreateProduct(ctx, input)
	case "propose_update_product":
		return t.proposeUpdateProduct(ctx, input)
	case "propose_stock_in":
		return t.proposeStock(ctx, input, actionStockIn)
	case "propose_stock_out":
		return t.proposeStock(ctx, input, actionStockOut)
	case "propose_transaction":
		return t.proposeTransaction(input)
	case "propose_market_check":
		return t.proposeMarketCheck(ctx, input)
	}
	return "", nil, toolError("ไม่มีเครื่องมือชื่อ " + name)
}

// ---------- อ่านข้อมูล ----------

func (t toolbox) shopInfo(ctx context.Context) (string, *Action, error) {
	shop, err := t.shop.Company(ctx)
	if err != nil {
		return "", nil, err
	}
	fields, err := t.shop.Fields(ctx)
	if err != nil {
		return "", nil, err
	}
	categories, err := t.categoryNames(ctx)
	if err != nil {
		return "", nil, err
	}
	return jsonText(map[string]any{
		"name":               shop.Name,
		"vat_registered":     shop.VATRegistered,
		"prices_include_vat": shop.PricesIncludeVAT,
		"use_cost_page":      shop.UseCostPage,
		"track_delivery":     shop.TrackDelivery,
		"product_fields":     fields,
		"product_categories": categories,
	})
}

// categoryNames ชื่อหมวดหมู่สินค้าทั้งหมด (ให้ AI เลือกใช้)
func (t toolbox) categoryNames(ctx context.Context) ([]string, error) {
	categories, err := t.shop.Categories(ctx)
	if err != nil {
		return nil, err
	}
	names := []string{}
	for _, c := range categories {
		names = append(names, c.Name)
	}
	return names, nil
}

// categoryID หา id ของหมวดจากชื่อที่ AI ส่งมา ("" = ไม่มีหมวด) ไม่มีหมวดชื่อนี้ → ส่งกลับให้ AI เลือกหมวดที่มี
// AI สร้างหมวดใหม่เองไม่ได้ ผู้ใช้เพิ่มที่เมนู สินค้า → หมวดหมู่สินค้า
func (t toolbox) categoryID(ctx context.Context, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil
	}
	categories, err := t.shop.Categories(ctx)
	if err != nil {
		return "", err
	}
	names := []string{}
	for _, c := range categories {
		if strings.EqualFold(strings.TrimSpace(c.Name), name) {
			return c.ID, nil
		}
		names = append(names, c.Name)
	}
	return "", toolError(fmt.Sprintf("ยังไม่มีหมวด %q ใช้หมวดที่มี: %v หรือเว้นว่าง (ถ้าต้องการหมวดใหม่ บอกผู้ใช้ให้เพิ่มที่เมนู สินค้า → หมวดหมู่สินค้า ก่อน)", name, names))
}

// listProducts สินค้าทั้งหมด ช่องข้อมูลเพิ่มเติมแปลงเป็น "ชื่อช่อง: ค่า" ให้ AI อ่านง่าย
func (t toolbox) listProducts(ctx context.Context) (string, *Action, error) {
	products, err := t.shop.Products(ctx)
	if err != nil {
		return "", nil, err
	}
	fields, err := t.shop.Fields(ctx)
	if err != nil {
		return "", nil, err
	}
	items := []map[string]any{}
	for _, p := range products {
		items = append(items, map[string]any{
			"id": p.ID, "code": p.Code, "name": p.Name, "category": p.Category, "unit": p.Unit,
			"cost_price": p.CostPrice, "sell_price": p.SellPrice, "stock": p.Stock, "min_stock": p.MinStock,
			"track_stock": p.TrackStock, "fields": fieldValues(p, fields),
		})
	}
	return jsonText(items)
}

func (t toolbox) costSheet(ctx context.Context, input []byte) (string, *Action, error) {
	p, err := t.productOf(ctx, input)
	if err != nil {
		return "", nil, err
	}
	sheet, err := t.shop.CostSheet(ctx, p.ID)
	if err != nil {
		return "", nil, err
	}
	return jsonText(sheet)
}

func (t toolbox) marketCheck(ctx context.Context, input []byte) (string, *Action, error) {
	p, err := t.productOf(ctx, input)
	if err != nil {
		return "", nil, err
	}
	check, err := t.repo.LatestMarketCheck(ctx, p.ID)
	if errors.Is(err, ErrNotFound) {
		return "สินค้านี้ยังไม่เคยเช็คราคาตลาด เสนอ propose_market_check ได้", nil, nil
	}
	if err != nil {
		return "", nil, err
	}
	return jsonText(check)
}

func (t toolbox) dashboard(ctx context.Context, input []byte) (string, *Action, error) {
	monthStart, err := t.monthOf(input)
	if err != nil {
		return "", nil, err
	}
	d, err := t.shop.Dashboard(ctx, monthStart)
	if err != nil {
		return "", nil, err
	}
	return jsonText(d)
}

func (t toolbox) transactions(ctx context.Context, input []byte) (string, *Action, error) {
	monthStart, err := t.monthOf(input)
	if err != nil {
		return "", nil, err
	}
	list, err := t.shop.Transactions(ctx, monthStart)
	if err != nil {
		return "", nil, err
	}
	return jsonText(list)
}

// ---------- เสนองาน (ตรวจข้อมูลด้วยกฎเดียวกับ API เส้นจริง ผิด → ส่งกลับให้ AI แก้) ----------

// productProposal ข้อมูลสินค้าที่เสนอ = สิ่งที่ POST / PUT /products รับ + ชื่อหมวดไว้โชว์ในการ์ด (API ไม่ใช้ field category)
type productProposal struct {
	product.ProductRequest
	Category string `json:"category"`
}

func (t toolbox) proposeCreateProduct(ctx context.Context, input []byte) (string, *Action, error) {
	var in struct {
		Name        string  `json:"name"`
		Unit        string  `json:"unit"`
		Category    string  `json:"category"`
		Code        string  `json:"code"`
		Description string  `json:"description"`
		SellPrice   float64 `json:"sell_price"`
		CostPrice   float64 `json:"cost_price"`
		MinStock    int     `json:"min_stock"`
		TrackStock  *bool   `json:"track_stock"`
		Reason      string  `json:"reason"`
	}
	if err := readInput(input, &in); err != nil {
		return "", nil, err
	}
	categoryID, err := t.categoryID(ctx, in.Category)
	if err != nil {
		return "", nil, err
	}
	trackStock := in.TrackStock == nil || *in.TrackStock
	req := product.ProductRequest{
		Name: in.Name, Unit: in.Unit, CategoryID: &categoryID, Code: in.Code, Description: in.Description,
		SellPrice: in.SellPrice, CostPrice: in.CostPrice, MinStock: in.MinStock,
		TrackStock: &trackStock, CustomFields: map[string]string{},
	}
	if code := req.Validate(); code != "" {
		return "", nil, toolError("ข้อมูลสินค้าไม่ถูกต้อง: " + code)
	}
	data := productProposal{ProductRequest: req, Category: strings.TrimSpace(in.Category)}
	return proposed(Action{Type: actionCreateProduct, Reason: in.Reason, ProductName: req.Name, Data: data})
}

// proposeUpdateProduct ส่งสินค้าทั้งตัวหลังแก้ (Flutter PUT ได้เลย) + รายการช่องที่เปลี่ยนไว้โชว์ผู้ใช้
func (t toolbox) proposeUpdateProduct(ctx context.Context, input []byte) (string, *Action, error) {
	var in struct {
		ProductID   string   `json:"product_id"`
		Name        *string  `json:"name"`
		Unit        *string  `json:"unit"`
		Category    *string  `json:"category"`
		Code        *string  `json:"code"`
		Description *string  `json:"description"`
		SellPrice   *float64 `json:"sell_price"`
		CostPrice   *float64 `json:"cost_price"`
		MinStock    *int     `json:"min_stock"`
		Reason      string   `json:"reason"`
	}
	if err := readInput(input, &in); err != nil {
		return "", nil, err
	}
	p, err := t.findProduct(ctx, in.ProductID)
	if err != nil {
		return "", nil, err
	}

	req := requestOf(p)
	categoryName := p.Category
	changes := []Change{}
	changeText(&changes, "name", &req.Name, in.Name)
	changeText(&changes, "unit", &req.Unit, in.Unit)
	if in.Category != nil && strings.TrimSpace(*in.Category) != p.Category {
		categoryID, err := t.categoryID(ctx, *in.Category)
		if err != nil {
			return "", nil, err
		}
		categoryName = strings.TrimSpace(*in.Category)
		changes = append(changes, Change{Field: "category", From: p.Category, To: categoryName})
		req.CategoryID = &categoryID
	}
	changeText(&changes, "code", &req.Code, in.Code)
	changeText(&changes, "description", &req.Description, in.Description)
	changeNumber(&changes, "sell_price", &req.SellPrice, in.SellPrice)
	changeNumber(&changes, "cost_price", &req.CostPrice, in.CostPrice)
	if in.MinStock != nil && *in.MinStock != req.MinStock {
		changes = append(changes, Change{Field: "min_stock", From: req.MinStock, To: *in.MinStock})
		req.MinStock = *in.MinStock
	}
	if len(changes) == 0 {
		return "", nil, toolError("ไม่มีช่องไหนเปลี่ยน ค่าที่ส่งมาเหมือนเดิมทั้งหมด")
	}
	if code := req.Validate(); code != "" {
		return "", nil, toolError("ข้อมูลสินค้าไม่ถูกต้อง: " + code)
	}
	return proposed(Action{
		Type: actionUpdateProduct, Reason: in.Reason, ProductID: p.ID, ProductName: p.Name, Changes: changes,
		Data: productProposal{ProductRequest: req, Category: categoryName},
	})
}

// proposeStock รับเข้า / เบิกออก — สินค้าต้องนับสต็อก, เบิกออกต้องมีของพอ
func (t toolbox) proposeStock(ctx context.Context, input []byte, actionType string) (string, *Action, error) {
	var in struct {
		ProductID string  `json:"product_id"`
		Quantity  int     `json:"quantity"`
		UnitCost  float64 `json:"unit_cost"`
		Supplier  string  `json:"supplier"`
		Note      string  `json:"note"`
		Reason    string  `json:"reason"`
	}
	if err := readInput(input, &in); err != nil {
		return "", nil, err
	}
	p, err := t.findProduct(ctx, in.ProductID)
	if err != nil {
		return "", nil, err
	}
	if !p.TrackStock {
		return "", nil, toolError("สินค้านี้ตั้งไว้ว่าไม่นับสต็อก รับเข้า / เบิกออกไม่ได้")
	}

	req := stock.MovementRequest{ProductID: p.ID, Quantity: in.Quantity, Note: in.Note}
	if actionType == actionStockIn {
		req.UnitCost = in.UnitCost
		req.Supplier = in.Supplier
	}
	if code := req.Validate(); code != "" {
		return "", nil, toolError("ข้อมูลไม่ถูกต้อง: " + code)
	}
	if actionType == actionStockOut && in.Quantity > p.Stock {
		return "", nil, toolError(fmt.Sprintf("ของไม่พอ คงเหลือแค่ %d %s", p.Stock, unitOf(p)))
	}
	return proposed(Action{Type: actionType, Reason: in.Reason, ProductID: p.ID, ProductName: p.Name, Data: req})
}

func (t toolbox) proposeTransaction(input []byte) (string, *Action, error) {
	var in struct {
		Type     string  `json:"type"`
		Category string  `json:"category"`
		Amount   float64 `json:"amount"`
		Date     string  `json:"date"`
		Note     string  `json:"note"`
		Reason   string  `json:"reason"`
	}
	if err := readInput(input, &in); err != nil {
		return "", nil, err
	}
	if in.Date == "" {
		in.Date = t.today.Format("2006-01-02")
	}
	req := transaction.TransactionRequest{Type: in.Type, Category: in.Category, Amount: in.Amount, Date: in.Date, Note: in.Note}
	if code := req.Validate(); code != "" {
		return "", nil, toolError("ข้อมูลไม่ถูกต้อง: " + code)
	}
	categories := expenseCategories
	if req.Type == "income" {
		categories = incomeCategories
	}
	if !slices.Contains(categories, req.Category) {
		return "", nil, toolError(fmt.Sprintf("หมวด %s ใช้กับ %s ไม่ได้ ใช้ได้: %v", req.Category, req.Type, categories))
	}
	return proposed(Action{Type: actionAddTransaction, Reason: in.Reason, Data: req})
}

func (t toolbox) proposeMarketCheck(ctx context.Context, input []byte) (string, *Action, error) {
	var in struct {
		ProductID string `json:"product_id"`
		Reason    string `json:"reason"`
	}
	if err := readInput(input, &in); err != nil {
		return "", nil, err
	}
	if !t.canSearch {
		return "", nil, toolError("AI ที่ใช้อยู่ค้นเว็บไม่ได้ เช็คราคาตลาดต้องใส่ key ของ Claude ที่หน้าตั้งค่า หรือให้ผู้ดูแลระบบตั้ง TAVILY_API_KEY (ค้นเว็บฟรี)")
	}
	p, err := t.findProduct(ctx, in.ProductID)
	if err != nil {
		return "", nil, err
	}
	return proposed(Action{
		Type: actionMarketCheck, Reason: in.Reason, ProductID: p.ID, ProductName: p.Name,
		Data: map[string]string{"product_id": p.ID},
	})
}

// ---------- ตัวช่วย ----------

// readInput อ่าน input ของเครื่องมือ อ่านไม่ได้ → ส่งกลับให้ AI แก้
func readInput(input []byte, target any) error {
	if err := json.Unmarshal(input, target); err != nil {
		return toolError("อ่านข้อมูลที่ส่งมาไม่ได้: " + err.Error())
	}
	return nil
}

// productOf อ่าน product_id จาก input แล้วหาสินค้า
func (t toolbox) productOf(ctx context.Context, input []byte) (product.Product, error) {
	var in struct {
		ProductID string `json:"product_id"`
	}
	if err := readInput(input, &in); err != nil {
		return product.Product{}, err
	}
	return t.findProduct(ctx, in.ProductID)
}

// findProduct หาสินค้า ไม่เจอ → บอก AI ให้ดู id จาก list_products
func (t toolbox) findProduct(ctx context.Context, id string) (product.Product, error) {
	notFound := toolError("ไม่เจอสินค้า id " + id + " ใช้ list_products ดู id ที่ถูกต้อง")
	if !utils.IsUUID(id) {
		return product.Product{}, notFound
	}
	p, err := t.shop.Product(ctx, id)
	if errors.Is(err, product.ErrNotFound) {
		return product.Product{}, notFound
	}
	return p, err
}

// monthOf อ่าน month (YYYY-MM) จาก input → วันแรกของเดือน ไม่ใส่ = เดือนนี้
func (t toolbox) monthOf(input []byte) (string, error) {
	var in struct {
		Month string `json:"month"`
	}
	if err := readInput(input, &in); err != nil {
		return "", err
	}
	if in.Month == "" {
		in.Month = t.today.Format("2006-01")
	}
	start, err := time.Parse("2006-01", in.Month)
	if err != nil {
		return "", toolError("month ต้องเป็นแบบ YYYY-MM เช่น 2026-10")
	}
	return start.Format("2006-01-02"), nil
}

// requestOf ข้อมูลสินค้าปัจจุบันในรูปแบบที่ PUT /products/{id} รับ
func requestOf(p product.Product) product.ProductRequest {
	trackStock := p.TrackStock
	categoryID := p.CategoryID
	customFields := p.CustomFields
	if customFields == nil {
		customFields = map[string]string{}
	}
	return product.ProductRequest{
		Code: p.Code, Name: p.Name, CategoryID: &categoryID, Description: p.Description, Unit: p.Unit,
		CostPrice: p.CostPrice, SellPrice: p.SellPrice, MinStock: p.MinStock,
		TrackStock: &trackStock, CustomFields: customFields,
	}
}

// changeText / changeNumber ถ้า AI ส่งค่าใหม่มาและไม่เหมือนเดิม → แก้ค่า + จดว่าเปลี่ยนอะไร
func changeText(changes *[]Change, field string, current *string, next *string) {
	if next == nil || *next == *current {
		return
	}
	*changes = append(*changes, Change{Field: field, From: *current, To: *next})
	*current = *next
}

func changeNumber(changes *[]Change, field string, current *float64, next *float64) {
	if next == nil || *next == *current {
		return
	}
	*changes = append(*changes, Change{Field: field, From: *current, To: *next})
	*current = *next
}

// proposed ตอบ AI ว่าเสนอแล้ว + คืนงานที่เสนอ (changes ว่าง → [] ใน JSON)
func proposed(action Action) (string, *Action, error) {
	if action.Changes == nil {
		action.Changes = []Change{}
	}
	return proposedText, &action, nil
}

// jsonText แปลงผลเป็น JSON ให้ AI อ่าน
func jsonText(value any) (string, *Action, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", nil, err
	}
	return string(data), nil, nil
}
