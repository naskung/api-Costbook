package helpers

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

const testPNG = "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"

// newUploadContext สร้าง request แบบ multipart ที่มีไฟล์ในช่อง field (data = nil → ไม่แนบไฟล์)
func newUploadContext(t *testing.T, field string, data []byte) *gin.Context {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if data != nil {
		part, err := writer.CreateFormFile(field, "photo.png")
		if err != nil {
			t.Fatal(err)
		}
		part.Write(data)
	}
	writer.Close()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPut, "/upload", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	return c
}

func TestReadImage(t *testing.T) {
	tests := []struct {
		name        string
		field       string
		data        []byte
		wantType    string
		wantProblem string
	}{
		{"รูป PNG ผ่าน", "image", []byte(testPNG), "image/png", ""},
		{"ไม่มีไฟล์ = required", "image", nil, "", "required"},
		{"ไฟล์อยู่ผิดช่อง = required", "photo", []byte(testPNG), "", "required"},
		{"ไฟล์ว่าง = required", "image", []byte{}, "", "required"},
		{"ไม่ใช่รูป = type_invalid", "image", []byte("hello"), "", "type_invalid"},
		{"เกิน 1 MB นิดเดียว = too_large", "image", append([]byte(testPNG), make([]byte, MaxImageSize)...), "", "too_large"},
		{"ใหญ่เกินมาก = too_large", "image", append([]byte(testPNG), make([]byte, 5*MaxImageSize)...), "", "too_large"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newUploadContext(t, tt.field, tt.data)
			data, contentType, problem := ReadImage(c, "image")
			if problem != tt.wantProblem || contentType != tt.wantType {
				t.Fatalf("ได้ (%q, %q), ต้องได้ (%q, %q)", contentType, problem, tt.wantType, tt.wantProblem)
			}
			if problem == "" && !bytes.Equal(data, tt.data) {
				t.Error("ไฟล์ที่อ่านได้ไม่ตรงกับที่ส่งไป")
			}
		})
	}
}
