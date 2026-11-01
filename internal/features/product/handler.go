package product

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/helpers"
)

type Handler struct {
	repo Repository
}

func NewHandler(repo Repository) *Handler {
	return &Handler{repo: repo}
}

// RegisterRoutes ต่อ route ของสินค้าเข้ากับกลุ่ม /api/v1 (ทุก route ต้อง login)
func (h *Handler) RegisterRoutes(api gin.IRouter, authRequired gin.HandlerFunc) {
	api.GET("/products", authRequired, h.List)
	api.GET("/products/counts", authRequired, h.Counts)
	api.GET("/products/:id", authRequired, h.Get)
	api.POST("/products", authRequired, h.Create)
	api.PUT("/products/:id", authRequired, h.Update)
	api.DELETE("/products/:id", authRequired, h.Delete)
	api.GET("/products/:id/image", authRequired, h.Image)
	api.PUT("/products/:id/image", authRequired, h.UploadImage)
	api.DELETE("/products/:id/image", authRequired, h.DeleteImage)
}

// List GET /api/v1/products?q=&category_id=&kind=sale|material&low=true&limit=20&offset=0
// ไม่ส่งอะไรเลย = สินค้าทั้งหมด (หน้าขาย / ต้นทุน / AI ใช้แบบนี้) / หน้ารายการสินค้าโหลดทีละหน้าพร้อมตัวกรอง
func (h *Handler) List(c *gin.Context) {
	limit, offset, ok := helpers.PageParams(c)
	if !ok {
		return
	}
	filter := ListFilter{
		Query:      c.Query("q"),
		CategoryID: c.Query("category_id"),
		Kind:       c.Query("kind"),
		OnlyLow:    c.Query("low") == "true",
		Limit:      limit,
		Offset:     offset,
	}
	if code := filter.Validate(); code != "" {
		helpers.WriteError(c, http.StatusBadRequest, code)
		return
	}
	products, err := h.repo.List(c.Request.Context(), filter)
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, products)
}

// Counts GET /api/v1/products/counts — จำนวนสินค้าทั้งร้าน (ทั้งหมด / สินค้าขาย / วัสดุ / ใกล้หมด)
func (h *Handler) Counts(c *gin.Context) {
	counts, err := h.repo.Counts(c.Request.Context())
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, counts)
}

// Get GET /api/v1/products/:id
func (h *Handler) Get(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "product_not_found")
	if !ok {
		return
	}

	product, err := h.repo.FindByID(c.Request.Context(), id)
	if errors.Is(err, ErrNotFound) {
		helpers.WriteError(c, http.StatusNotFound, "product_not_found")
		return
	}
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, product)
}

// Create POST /api/v1/products
func (h *Handler) Create(c *gin.Context) {
	req, ok := readRequest(c)
	if !ok {
		return
	}

	product, err := h.repo.Create(c.Request.Context(), req)
	if errors.Is(err, ErrCodeTaken) {
		helpers.WriteError(c, http.StatusConflict, "product_code_taken")
		return
	}
	if errors.Is(err, ErrCategoryNotFound) {
		helpers.WriteError(c, http.StatusBadRequest, "product_category_invalid")
		return
	}
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusCreated, product)
}

// Update PUT /api/v1/products/:id
func (h *Handler) Update(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "product_not_found")
	if !ok {
		return
	}
	req, ok := readRequest(c)
	if !ok {
		return
	}

	product, err := h.repo.Update(c.Request.Context(), id, req)
	if errors.Is(err, ErrNotFound) {
		helpers.WriteError(c, http.StatusNotFound, "product_not_found")
		return
	}
	if errors.Is(err, ErrCodeTaken) {
		helpers.WriteError(c, http.StatusConflict, "product_code_taken")
		return
	}
	if errors.Is(err, ErrCategoryNotFound) {
		helpers.WriteError(c, http.StatusBadRequest, "product_category_invalid")
		return
	}
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, product)
}

// Delete DELETE /api/v1/products/:id
func (h *Handler) Delete(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "product_not_found")
	if !ok {
		return
	}

	err := h.repo.Delete(c.Request.Context(), id)
	if errors.Is(err, ErrNotFound) {
		helpers.WriteError(c, http.StatusNotFound, "product_not_found")
		return
	}
	if errors.Is(err, ErrInUse) {
		helpers.WriteError(c, http.StatusConflict, "product_in_use")
		return
	}
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// readRequest อ่าน JSON + Validate() ไม่ผ่าน → ตอบ 400 แล้วคืน ok = false
func readRequest(c *gin.Context) (ProductRequest, bool) {
	var req ProductRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		helpers.WriteError(c, http.StatusBadRequest, "invalid_json")
		return req, false
	}

	code := req.Validate()
	if code != "" {
		helpers.WriteError(c, http.StatusBadRequest, code)
		return req, false
	}
	return req, true
}

// Image GET /api/v1/products/:id/image — ตอบเป็นรูปตรง ๆ
// URL มี ?v=<image_version> เปลี่ยนรูปแล้ว URL เปลี่ยน → ให้ browser เก็บรูปไว้ได้นาน ไม่ต้องโหลดซ้ำ
func (h *Handler) Image(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "product_not_found")
	if !ok {
		return
	}

	image, err := h.repo.FindImage(c.Request.Context(), id)
	if errors.Is(err, ErrNotFound) {
		helpers.WriteError(c, http.StatusNotFound, "product_not_found")
		return
	}
	if errors.Is(err, ErrImageNotFound) {
		helpers.WriteError(c, http.StatusNotFound, "product_image_not_found")
		return
	}
	if err != nil {
		helpers.ServerError(c, err)
		return
	}

	c.Header("Cache-Control", "private, max-age=31536000, immutable")
	c.Header("X-Content-Type-Options", "nosniff") // บอก browser ว่าเป็นรูปเท่านั้น ห้ามเดาเป็นชนิดอื่น
	c.Data(http.StatusOK, image.ContentType, image.Data)
}

// UploadImage PUT /api/v1/products/:id/image — ส่งแบบ multipart/form-data ช่องชื่อ "image" ได้สินค้าที่มี image_version ใหม่
func (h *Handler) UploadImage(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "product_not_found")
	if !ok {
		return
	}

	data, contentType, problem := helpers.ReadImage(c, "image")
	if problem != "" {
		helpers.WriteError(c, http.StatusBadRequest, "product_image_"+problem)
		return
	}

	product, err := h.repo.SaveImage(c.Request.Context(), id, Image{Data: data, ContentType: contentType})
	h.writeProduct(c, product, err)
}

// DeleteImage DELETE /api/v1/products/:id/image — ได้สินค้าที่ image_version = 0
func (h *Handler) DeleteImage(c *gin.Context) {
	id, ok := helpers.UUIDParam(c, "id", "product_not_found")
	if !ok {
		return
	}

	product, err := h.repo.DeleteImage(c.Request.Context(), id)
	h.writeProduct(c, product, err)
}

// writeProduct ตอบสินค้า 200 / ไม่เจอ 404 / DB พัง 500
func (h *Handler) writeProduct(c *gin.Context, product Product, err error) {
	if errors.Is(err, ErrNotFound) {
		helpers.WriteError(c, http.StatusNotFound, "product_not_found")
		return
	}
	if err != nil {
		helpers.ServerError(c, err)
		return
	}
	c.JSON(http.StatusOK, product)
}
