package barcode

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/helpers"
)

type Handler struct {
	finder Finder
}

func NewHandler(finder Finder) *Handler {
	return &Handler{finder: finder}
}

// RegisterRoutes ต่อ route หาข้อมูลสินค้าจากบาร์โค้ดเข้ากับกลุ่ม /api/v1 (ต้อง login)
func (h *Handler) RegisterRoutes(api gin.IRouter, authRequired gin.HandlerFunc) {
	api.GET("/barcodes/:code", authRequired, h.Get)
}

// Get GET /api/v1/barcodes/8850124003850 → ชื่อ / ยี่ห้อ / ขนาด ไว้กรอกฟอร์มสินค้าให้
// เลขไม่ใช่บาร์โค้ด 400 / ไม่เจอ 404 / ฐานข้อมูลภายนอกล่ม หรือเน็ตหลุด 502
func (h *Handler) Get(c *gin.Context) {
	code := c.Param("code")
	if !isBarcode(code) {
		helpers.WriteError(c, http.StatusBadRequest, "barcode_invalid")
		return
	}

	product, err := h.finder.Find(c.Request.Context(), code)
	if errors.Is(err, ErrNotFound) {
		helpers.WriteError(c, http.StatusNotFound, "barcode_not_found")
		return
	}
	if err != nil {
		// ไม่ใช่ความผิดของเรา / ผู้ใช้ → แจ้งว่าหาไม่ได้ตอนนี้ (log error ไว้ดู แต่ไม่ส่งรายละเอียดให้ client)
		slog.Error("หาข้อมูลบาร์โค้ดจาก Open Food Facts ไม่สำเร็จ", "code", code, "error", err)
		helpers.WriteError(c, http.StatusBadGateway, "barcode_lookup_failed")
		return
	}
	c.JSON(http.StatusOK, product)
}
