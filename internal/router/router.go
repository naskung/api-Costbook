package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/naskung/api-Costbook/docs"
	"github.com/naskung/api-Costbook/internal/config"
	"github.com/naskung/api-Costbook/internal/features/auth"
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

	return r
}

// health GET /api/v1/health เช็คว่า API ทำงานอยู่
func health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"}) // gin.H คือ map ไว้สร้าง JSON สั้น ๆ
}
