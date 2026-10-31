package company

import "testing"

// ไบต์แรก ๆ ของไฟล์แต่ละชนิด (ใช้แทนไฟล์รูปจริงใน test)
const (
	pngBytes  = "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"
	jpegBytes = "\xff\xd8\xff\xe0\x00\x10JFIF\x00"
	webpBytes = "RIFF\x24\x00\x00\x00WEBPVP8 "
	gifBytes  = "GIF89a\x01\x00\x01\x00"
)

func TestLogoContentType(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{"PNG ผ่าน", pngBytes, "image/png"},
		{"JPG ผ่าน", jpegBytes, "image/jpeg"},
		{"WebP ผ่าน", webpBytes, "image/webp"},
		{"GIF ไม่รับ", gifBytes, ""},
		{"SVG ไม่รับ", `<svg xmlns="http://www.w3.org/2000/svg"></svg>`, ""},
		{"ไฟล์ข้อความ ไม่รับ", "hello", ""},
		{"ไฟล์ PDF ไม่รับ", "%PDF-1.7", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := logoContentType([]byte(tt.data))
			if got != tt.want {
				t.Errorf("logoContentType() = %q, ต้องได้ %q", got, tt.want)
			}
		})
	}
}
