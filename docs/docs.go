package docs

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

// go:embed ฝังไฟล์เข้าไปในโปรแกรมตอน build → เปิด server จาก folder ไหนก็หาไฟล์เจอ
// (Go บังคับให้เป็น var แต่เราไม่แก้ค่าเลย จึงไม่ผิดกฎห้ามใช้ global ที่เปลี่ยนค่าได้)

//go:embed openapi.yaml
var openAPISpec []byte

//go:embed swagger.html
var swaggerPage []byte

// RegisterRoutes เปิดหน้าเอกสาร API (Swagger)
// GET /docs              หน้าเว็บดูเอกสาร + กดลองเรียก API ได้
// GET /docs/openapi.yaml ไฟล์เอกสาร API
func RegisterRoutes(r gin.IRouter) {
	r.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", swaggerPage)
	})
	r.GET("/docs/openapi.yaml", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", openAPISpec)
	})
}
