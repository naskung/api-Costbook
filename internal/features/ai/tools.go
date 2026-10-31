package ai

import (
	"sort"

	"github.com/anthropics/anthropic-sdk-go/v2"
)

// ชนิดของงานที่ AI เสนอได้ (ดู Action ใน model.go)
const (
	actionCreateProduct  = "create_product"
	actionUpdateProduct  = "update_product"
	actionStockIn        = "stock_in"
	actionStockOut       = "stock_out"
	actionAddTransaction = "add_transaction"
	actionMarketCheck    = "market_check"
)

// หมวดรายรับ-รายจ่าย (ตรงกับ TransactionModel ฝั่ง Flutter)
var (
	incomeCategories  = []string{"sales", "other_income"}
	expenseCategories = []string{"material", "labor", "rent", "utilities", "shipping", "marketing", "other_expense"}
)

// ---------- ตัวช่วยเขียน JSON Schema ของเครื่องมือ ----------

func stringProp(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func numberProp(description string) map[string]any {
	return map[string]any{"type": "number", "description": description}
}

func integerProp(description string) map[string]any {
	return map[string]any{"type": "integer", "description": description}
}

func boolProp(description string) map[string]any {
	return map[string]any{"type": "boolean", "description": description}
}

func enumProp(description string, values []string) map[string]any {
	return map[string]any{"type": "string", "enum": values, "description": description}
}

// objectProp object ซ้อนในเครื่องมือแบบ strict: ทุกช่องต้องอยู่ใน required และห้ามมีช่องอื่น
func objectProp(properties map[string]any) map[string]any {
	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             keysOf(properties),
		"additionalProperties": false,
	}
}

func arrayProp(description string, item map[string]any) map[string]any {
	return map[string]any{"type": "array", "description": description, "items": item}
}

// keysOf ชื่อทุกช่อง (เรียงตามตัวอักษร → schema เหมือนเดิมทุกครั้ง cache ได้)
func keysOf(properties map[string]any) []string {
	keys := make([]string, 0, len(properties))
	for key := range properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// toolSpec เครื่องมือของแชท 1 ตัว เขียนครั้งเดียว ใช้ได้ทั้ง Claude และ Gemini
// required = ช่องที่ต้องส่ง
type toolSpec struct {
	name        string
	description string
	properties  map[string]any
	required    []string
}

func newTool(name, description string, properties map[string]any, required ...string) toolSpec {
	return toolSpec{name: name, description: description, properties: properties, required: required}
}

// claudeTools แปลงเป็นรูปแบบเครื่องมือของ Claude
func claudeTools(specs []toolSpec) []anthropic.BetaToolUnionParam {
	tools := []anthropic.BetaToolUnionParam{}
	for _, spec := range specs {
		tools = append(tools, anthropic.BetaToolUnionParam{OfTool: &anthropic.BetaToolParam{
			Name:        spec.name,
			Description: anthropic.String(spec.description),
			InputSchema: anthropic.BetaToolInputSchemaParam{Properties: spec.properties, Required: spec.required},
		}})
	}
	return tools
}

// geminiTools แปลงเป็น functionDeclarations ของ Gemini (เครื่องมือที่ไม่มีช่องให้กรอก ไม่ต้องส่ง parameters)
func geminiTools(specs []toolSpec) []map[string]any {
	declarations := []map[string]any{}
	for _, spec := range specs {
		declaration := map[string]any{"name": spec.name, "description": spec.description}
		if len(spec.properties) > 0 {
			parameters := map[string]any{"type": "object", "properties": spec.properties}
			if len(spec.required) > 0 {
				parameters["required"] = spec.required
			}
			declaration["parameters"] = parameters
		}
		declarations = append(declarations, declaration)
	}
	return []map[string]any{{"functionDeclarations": declarations}}
}

// newStrictTool เครื่องมือที่ Claude ต้องส่งข้อมูลตรงตาม schema เป๊ะ (strict) — ใช้กับผลเช็คราคาตลาด
func newStrictTool(name, description string, properties map[string]any) anthropic.BetaToolUnionParam {
	return anthropic.BetaToolUnionParam{OfTool: &anthropic.BetaToolParam{
		Name:        name,
		Description: anthropic.String(description),
		Strict:      anthropic.Bool(true),
		InputSchema: anthropic.BetaToolInputSchemaParam{
			Properties:  properties,
			Required:    keysOf(properties),
			ExtraFields: map[string]any{"additionalProperties": false},
		},
	}}
}

// ---------- เครื่องมือของงานเช็คราคาตลาด ----------

const reportToolName = "save_market_report"

// reportTool Claude เรียกตอนจบเพื่อส่งผลเป็นข้อมูลที่เก็บลง DB ได้เลย
var reportTool = newStrictTool(reportToolName, "บันทึกผลสำรวจราคาตลาดของสินค้า เรียกครั้งเดียวตอนค้นเสร็จ ราคาเป็นบาทต่อหน่วยของร้าน 0 = ไม่รู้", map[string]any{
	"summary":        stringProp("สรุปภาพรวมราคาในตลาด 2–4 ประโยค"),
	"retail_low":     numberProp("ราคาขายปลีกต่ำสุดที่เจอ"),
	"retail_high":    numberProp("ราคาขายปลีกสูงสุดที่เจอ"),
	"retail_typical": numberProp("ราคาขายปลีกที่เจอบ่อยที่สุด"),
	"wholesale_low":  numberProp("ราคาขายส่งต่ำสุดต่อหน่วย"),
	"wholesale_high": numberProp("ราคาขายส่งสูงสุดต่อหน่วย"),
	"wholesale_note": stringProp("เงื่อนไขขายส่ง เช่น ขั้นต่ำ 50 ชิ้น หรือบอกว่าไม่เจอราคาส่ง"),
	"market_cost":    numberProp("ต้นทุนต่อหน่วยโดยประมาณของคนที่ทำขาย"),
	"cost_items": arrayProp("ต้นทุนแยกรายการ ต่อหน่วย", objectProp(map[string]any{
		"name":   stringProp("ชื่อรายการ เช่น วัตถุดิบ ค่าแรง ค่าแพ็ก"),
		"amount": numberProp("บาทต่อหน่วย"),
		"note":   stringProp("คิดจากอะไร"),
	})),
	"suggested_price": numberProp("ราคาขายที่แนะนำ (ราคาที่ลูกค้าจ่าย)"),
	"advice":          stringProp("คำแนะนำสั้น ๆ ที่ทำได้จริง"),
	"confidence":      enumProp("ข้อมูลที่เจอเชื่อถือได้แค่ไหน", []string{confidenceLow, confidenceMid, confidenceHigh}),
	"sources": arrayProp("ลิงก์ที่ใช้จริง 3–8 รายการ", objectProp(map[string]any{
		"title": stringProp("ชื่อร้าน / หน้าเว็บ"),
		"url":   stringProp("ลิงก์"),
		"price": numberProp("ราคาที่เห็น บาทต่อหน่วยของร้าน"),
		"kind":  enumProp("ชนิดราคา", []string{sourceRetail, sourceWholesale, sourceMaterial}),
	})),
})

// webSearchTool ค้นเว็บ (Anthropic ค้นให้) จำกัด 8 ครั้งต่องาน + บอกว่าอยู่ไทย ผลค้นหาจะเป็นของไทย
var webSearchTool = anthropic.BetaToolUnionParam{OfWebSearchTool20260209: &anthropic.BetaWebSearchTool20260209Param{
	MaxUses: anthropic.Int(8),
	UserLocation: anthropic.BetaUserLocationParam{
		Country:  anthropic.String("TH"),
		City:     anthropic.String("Bangkok"),
		Timezone: anthropic.String("Asia/Bangkok"),
	},
}}

// ---------- เครื่องมือของแชท ----------

var productIDProp = stringProp("id ของสินค้า (ดูจาก list_products)")
var reasonProp = stringProp("เหตุผลสั้น ๆ ว่าทำไมเสนองานนี้ (ผู้ใช้เห็นข้อความนี้)")

// chatTools อ่านข้อมูล (get_* / list_*) ทำงานทันที, เสนองาน (propose_*) ต้องให้ผู้ใช้กดยืนยัน
var chatTools = []toolSpec{
	newTool("get_shop_info", "ข้อมูลร้าน: ชื่อ จด VAT ไหม ราคาขายรวม VAT ไหม และช่องข้อมูลเพิ่มเติมของสินค้า", map[string]any{}),
	newTool("list_products", "สินค้าทั้งหมด: id รหัส ชื่อ หมวด หน่วย ต้นทุน ราคาขาย คงเหลือ ขั้นต่ำ นับสต็อกไหม ช่องเพิ่มเติม", map[string]any{}),
	newTool("get_cost_sheet", "ใบต้นทุนของสินค้า: รายการต้นทุน จำนวนผลิตต่อรอบ ต้นทุนต่อหน่วย", map[string]any{
		"product_id": productIDProp,
	}, "product_id"),
	newTool("get_market_check", "ผลเช็คราคาตลาดครั้งล่าสุดของสินค้า (ที่เคยเช็คไว้ ไม่ได้ค้นใหม่)", map[string]any{
		"product_id": productIDProp,
	}, "product_id"),
	newTool("get_dashboard", "ภาพรวมของเดือน: รายรับ รายจ่าย กำไร ยอดขาย กำไรขั้นต้น สินค้าขายดี ยอดตามหมวด ย้อนหลัง 6 เดือน มูลค่าสต็อก เทียบราคาตลาด", map[string]any{
		"month": stringProp("เดือนแบบ YYYY-MM ไม่ใส่ = เดือนนี้"),
	}),
	newTool("list_transactions", "รายการรายรับ-รายจ่ายของเดือน", map[string]any{
		"month": stringProp("เดือนแบบ YYYY-MM ไม่ใส่ = เดือนนี้"),
	}),

	newTool("propose_create_product", "เสนอเพิ่มสินค้าใหม่ (ผู้ใช้ต้องกดยืนยัน)", map[string]any{
		"name":        stringProp("ชื่อสินค้า"),
		"unit":        stringProp("หน่วย เช่น ชิ้น กล่อง"),
		"category":    stringProp("หมวด"),
		"code":        stringProp("รหัส / บาร์โค้ด (ห้ามซ้ำ ไม่มีให้เว้นว่าง)"),
		"description": stringProp("รายละเอียด"),
		"sell_price":  numberProp("ราคาขายต่อหน่วย"),
		"cost_price":  numberProp("ต้นทุนต่อหน่วย"),
		"min_stock":   integerProp("จำนวนขั้นต่ำก่อนเตือนใกล้หมด"),
		"track_stock": boolProp("นับสต็อกไหม (งานบริการ / สั่งทำ = false)"),
		"reason":      reasonProp,
	}, "name", "sell_price", "reason"),
	newTool("propose_update_product", "เสนอแก้ข้อมูลสินค้า ส่งมาเฉพาะช่องที่จะเปลี่ยน (ผู้ใช้ต้องกดยืนยัน)", map[string]any{
		"product_id":  productIDProp,
		"name":        stringProp("ชื่อใหม่"),
		"unit":        stringProp("หน่วยใหม่"),
		"category":    stringProp("หมวดใหม่"),
		"code":        stringProp("รหัสใหม่"),
		"description": stringProp("รายละเอียดใหม่"),
		"sell_price":  numberProp("ราคาขายใหม่"),
		"cost_price":  numberProp("ต้นทุนใหม่"),
		"min_stock":   integerProp("จำนวนขั้นต่ำใหม่"),
		"reason":      reasonProp,
	}, "product_id", "reason"),
	newTool("propose_stock_in", "เสนอรับสินค้าเข้าสต็อก (ผู้ใช้ต้องกดยืนยัน) ใส่ unit_cost แล้วระบบคิดต้นทุนเฉลี่ยให้", map[string]any{
		"product_id": productIDProp,
		"quantity":   integerProp("จำนวนที่รับเข้า"),
		"unit_cost":  numberProp("ต้นทุนต่อหน่วยของล็อตนี้ (ไม่รู้ = 0)"),
		"supplier":   stringProp("ซื้อจากใคร"),
		"note":       stringProp("หมายเหตุ"),
		"reason":     reasonProp,
	}, "product_id", "quantity", "reason"),
	newTool("propose_stock_out", "เสนอเบิกสินค้าออกจากสต็อก เช่น ของเสีย ใช้เอง (ผู้ใช้ต้องกดยืนยัน) — การขายให้ออกเอกสารขายแทน", map[string]any{
		"product_id": productIDProp,
		"quantity":   integerProp("จำนวนที่เบิกออก"),
		"note":       stringProp("หมายเหตุ"),
		"reason":     reasonProp,
	}, "product_id", "quantity", "reason"),
	newTool("propose_transaction", "เสนอบันทึกรายรับ-รายจ่าย (ผู้ใช้ต้องกดยืนยัน) รายรับจากการขายที่ออกเอกสารขายแล้ว ระบบบันทึกให้เอง ห้ามบันทึกซ้ำ", map[string]any{
		"type":     enumProp("income = รายรับ / expense = รายจ่าย", []string{"income", "expense"}),
		"category": enumProp("หมวด: รายรับ = sales, other_income / รายจ่าย = material (วัตถุดิบ), labor (ค่าแรง), rent (ค่าเช่า), utilities (ค่าน้ำค่าไฟ), shipping (ค่าส่ง), marketing (การตลาด), other_expense", append(append([]string{}, incomeCategories...), expenseCategories...)),
		"amount":   numberProp("จำนวนเงิน"),
		"date":     stringProp("วันที่ YYYY-MM-DD ไม่ใส่ = วันนี้"),
		"note":     stringProp("หมายเหตุ"),
		"reason":   reasonProp,
	}, "type", "category", "amount", "reason"),
	newTool("propose_market_check", "เสนอให้ AI ค้นราคาตลาดของสินค้าใหม่ (มีค่าใช้จ่าย ผู้ใช้ต้องกดยืนยัน)", map[string]any{
		"product_id": productIDProp,
		"reason":     reasonProp,
	}, "product_id", "reason"),
}
