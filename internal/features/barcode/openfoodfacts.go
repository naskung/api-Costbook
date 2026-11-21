package barcode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Open Food Facts = ฐานข้อมูลสินค้าฟรีที่คนทั่วโลกช่วยกันเพิ่ม (ไม่ต้องมี key) ส่วนใหญ่เป็นอาหาร / เครื่องดื่ม / ของใช้
// สินค้าไทยมีประมาณ 11,000 รายการ (2026-11) หาไม่เจอก็กรอกเองเหมือนเดิม
const openFoodFactsBaseURL = "https://world.openfoodfacts.org"

// openFoodFactsUserAgent Open Food Facts ขอให้บอกชื่อแอปที่เรียก (ไม่ใส่ข้อมูลส่วนตัวของร้าน)
const openFoodFactsUserAgent = "CostBook/1.0"

// openFoodFactsTimeout รอนานสุดเท่านี้ (ผู้ใช้รออยู่หน้าฟอร์ม) เกินแล้วบอกว่าหาไม่ได้
const openFoodFactsTimeout = 8 * time.Second

// ErrNotFound ไม่มีสินค้านี้ในฐานข้อมูล หรือมีแต่ไม่มีชื่อ (handler ตอบ 404)
var ErrNotFound = errors.New("barcode not found")

// Finder หาข้อมูลสินค้าจากเลขบาร์โค้ด (แยกเป็น interface เพื่อให้ test ใช้ตัวปลอมได้)
type Finder interface {
	Find(ctx context.Context, code string) (Product, error)
}

type openFoodFacts struct {
	baseURL string
	http    *http.Client
}

func NewOpenFoodFacts() Finder {
	return &openFoodFacts{baseURL: openFoodFactsBaseURL, http: &http.Client{Timeout: openFoodFactsTimeout}}
}

// offResponse คำตอบของ GET /api/v2/product/{code} (เอาเฉพาะช่องที่ใช้) status 1 = เจอ, 0 = ไม่เจอ
type offResponse struct {
	Status  int `json:"status"`
	Product struct {
		NameTH   string `json:"product_name_th"`
		Name     string `json:"product_name"`
		NameEN   string `json:"product_name_en"`
		Brands   string `json:"brands"`
		Quantity string `json:"quantity"`
	} `json:"product"`
}

// Find ถาม Open Food Facts ด้วยเลขบาร์โค้ด — ไม่เจอ = ErrNotFound / ต่อไม่ได้ หรือตอบผิดปกติ = error อื่น
func (o *openFoodFacts) Find(ctx context.Context, code string) (Product, error) {
	url := o.baseURL + "/api/v2/product/" + code + "?fields=product_name_th,product_name,product_name_en,brands,quantity"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Product{}, err
	}
	req.Header.Set("User-Agent", openFoodFactsUserAgent)

	resp, err := o.http.Do(req)
	if err != nil {
		return Product{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Product{}, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return Product{}, fmt.Errorf("open food facts status %d", resp.StatusCode)
	}

	var body offResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return Product{}, err
	}
	return productFrom(code, body)
}

// productFrom แปลงคำตอบเป็น Product — ไม่เจอ / ไม่มีชื่อเลย = ErrNotFound (ไม่มีอะไรให้กรอก)
func productFrom(code string, body offResponse) (Product, error) {
	name := firstNonEmpty(body.Product.NameTH, body.Product.Name, body.Product.NameEN)
	if body.Status != 1 || name == "" {
		return Product{}, ErrNotFound
	}
	return Product{
		Code:     code,
		Name:     name,
		Brand:    firstBrand(body.Product.Brands),
		Quantity: firstNonEmpty(body.Product.Quantity),
	}, nil
}
