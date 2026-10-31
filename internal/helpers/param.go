package helpers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/utils"
)

// UUIDParam อ่าน id จาก path เช่น /products/:id
// id ไม่ใช่ UUID → ตอบ 404 ด้วย code ที่ให้มา แล้วคืน ok = false (handler return ได้เลย)
func UUIDParam(c *gin.Context, name, notFoundCode string) (string, bool) {
	id := c.Param(name)
	if !utils.IsUUID(id) {
		WriteError(c, http.StatusNotFound, notFoundCode)
		return "", false
	}
	return id, true
}
