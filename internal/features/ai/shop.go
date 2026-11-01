package ai

import (
	"context"

	"github.com/naskung/api-Costbook/internal/features/category"
	"github.com/naskung/api-Costbook/internal/features/company"
	"github.com/naskung/api-Costbook/internal/features/cost"
	"github.com/naskung/api-Costbook/internal/features/dashboard"
	"github.com/naskung/api-Costbook/internal/features/field"
	"github.com/naskung/api-Costbook/internal/features/product"
	"github.com/naskung/api-Costbook/internal/features/transaction"
)

// ShopReader ข้อมูลร้านที่ AI อ่านได้ (อ่านอย่างเดียว — การแก้ข้อมูลต้องให้ผู้ใช้กดยืนยันใน Flutter)
// ใช้ repository ของ feature อื่น ไม่เขียน SQL ซ้ำ
type ShopReader interface {
	Company(ctx context.Context) (company.Company, error)
	Fields(ctx context.Context) ([]field.ProductField, error)
	Categories(ctx context.Context) ([]category.Category, error)
	Products(ctx context.Context) ([]product.Product, error)
	Product(ctx context.Context, id string) (product.Product, error)
	CostSheet(ctx context.Context, productID string) (cost.CostSheet, error)
	Dashboard(ctx context.Context, monthStart string) (dashboard.Dashboard, error)
	Transactions(ctx context.Context, monthStart string) ([]transaction.Transaction, error)
}

// Shop ต่อ ShopReader เข้ากับ repository จริงของแต่ละ feature
type Shop struct {
	CompanyRepo     company.Repository
	FieldRepo       field.Repository
	CategoryRepo    category.Repository
	ProductRepo     product.Repository
	CostRepo        cost.Repository
	DashboardRepo   dashboard.Repository
	TransactionRepo transaction.Repository
}

func (s Shop) Company(ctx context.Context) (company.Company, error) {
	return s.CompanyRepo.Get(ctx)
}

func (s Shop) Fields(ctx context.Context) ([]field.ProductField, error) {
	return s.FieldRepo.List(ctx)
}

func (s Shop) Categories(ctx context.Context) ([]category.Category, error) {
	return s.CategoryRepo.List(ctx, 0, 0)
}

func (s Shop) Products(ctx context.Context) ([]product.Product, error) {
	return s.ProductRepo.List(ctx, product.ListFilter{})
}

// Product ไม่เจอสินค้า → product.ErrNotFound
func (s Shop) Product(ctx context.Context, id string) (product.Product, error) {
	return s.ProductRepo.FindByID(ctx, id)
}

func (s Shop) CostSheet(ctx context.Context, productID string) (cost.CostSheet, error) {
	return s.CostRepo.Get(ctx, productID)
}

func (s Shop) Dashboard(ctx context.Context, monthStart string) (dashboard.Dashboard, error) {
	return s.DashboardRepo.Get(ctx, monthStart)
}

func (s Shop) Transactions(ctx context.Context, monthStart string) ([]transaction.Transaction, error) {
	return s.TransactionRepo.List(ctx, monthStart, 0, 0)
}
