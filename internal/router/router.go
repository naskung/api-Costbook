package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/naskung/api-Costbook/docs"
	"github.com/naskung/api-Costbook/internal/config"
	"github.com/naskung/api-Costbook/internal/features/ai"
	"github.com/naskung/api-Costbook/internal/features/auth"
	"github.com/naskung/api-Costbook/internal/features/category"
	"github.com/naskung/api-Costbook/internal/features/company"
	"github.com/naskung/api-Costbook/internal/features/cost"
	"github.com/naskung/api-Costbook/internal/features/dashboard"
	"github.com/naskung/api-Costbook/internal/features/field"
	"github.com/naskung/api-Costbook/internal/features/invoice"
	"github.com/naskung/api-Costbook/internal/features/product"
	"github.com/naskung/api-Costbook/internal/features/stock"
	"github.com/naskung/api-Costbook/internal/features/transaction"
	"github.com/naskung/api-Costbook/internal/helpers"
	"github.com/naskung/api-Costbook/internal/middleware"
)

// New สร้าง router ของ API และต่อ route ทั้งหมดไว้ที่นี่ที่เดียว
func New(cfg config.Config, db *pgxpool.Pool) *gin.Engine {
	r := gin.Default() // มี Logger (พิมพ์ทุก request) + Recovery (โค้ด panic แล้ว server ไม่ล่ม) ให้แล้ว
	r.Use(middleware.CORS(cfg.AllowedOrigin))
	r.NoRoute(func(c *gin.Context) {
		helpers.WriteError(c, http.StatusNotFound, "not_found") // path ที่ไม่มี ก็ตอบ JSON แบบเดียวกับ error อื่น
	})

	docs.RegisterRoutes(r) // หน้าเอกสาร API: http://localhost:8080/docs

	api := r.Group("/api/v1")
	api.GET("/health", health)

	authRequired := middleware.Auth(cfg.JWTSecret)

	authHandler := auth.NewHandler(auth.NewPostgresRepository(db), cfg.JWTSecret)
	authHandler.RegisterRoutes(api, authRequired)

	companyRepo := company.NewPostgresRepository(db)
	companyHandler := company.NewHandler(companyRepo)
	companyHandler.RegisterRoutes(api, authRequired)

	productRepo := product.NewPostgresRepository(db)
	productHandler := product.NewHandler(productRepo)
	productHandler.RegisterRoutes(api, authRequired)

	fieldRepo := field.NewPostgresRepository(db)
	fieldHandler := field.NewHandler(fieldRepo)
	fieldHandler.RegisterRoutes(api, authRequired)

	categoryRepo := category.NewPostgresRepository(db)
	categoryHandler := category.NewHandler(categoryRepo)
	categoryHandler.RegisterRoutes(api, authRequired)

	costRepo := cost.NewPostgresRepository(db)
	costHandler := cost.NewHandler(costRepo)
	costHandler.RegisterRoutes(api, authRequired)

	stockHandler := stock.NewHandler(stock.NewPostgresRepository(db))
	stockHandler.RegisterRoutes(api, authRequired)

	transactionRepo := transaction.NewPostgresRepository(db)
	transactionHandler := transaction.NewHandler(transactionRepo)
	transactionHandler.RegisterRoutes(api, authRequired)

	invoiceHandler := invoice.NewHandler(invoice.NewPostgresRepository(db))
	invoiceHandler.RegisterRoutes(api, authRequired)

	dashboardRepo := dashboard.NewPostgresRepository(db)
	dashboardHandler := dashboard.NewHandler(dashboardRepo)
	dashboardHandler.RegisterRoutes(api, authRequired)

	// AI อ่านข้อมูลร้านผ่าน repository ของ feature อื่น / เลือก Claude (มี key) หรือ Gemini ฟรีให้เอง
	shop := ai.Shop{
		CompanyRepo: companyRepo, FieldRepo: fieldRepo, CategoryRepo: categoryRepo, ProductRepo: productRepo,
		CostRepo: costRepo, DashboardRepo: dashboardRepo, TransactionRepo: transactionRepo,
	}
	aiConfig := ai.Config{
		ClaudeKey: cfg.AnthropicAPIKey, GeminiKey: cfg.GeminiAPIKey, GeminiModel: cfg.GeminiModel, Secret: cfg.JWTSecret,
	}
	aiHandler := ai.NewHandler(ai.NewPostgresRepository(db), shop, aiConfig)
	aiHandler.RegisterRoutes(api, authRequired)

	return r
}

// health GET /api/v1/health เช็คว่า API ทำงานอยู่
func health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"}) // gin.H คือ map ไว้สร้าง JSON สั้น ๆ
}
