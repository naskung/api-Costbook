package utils

import "testing"

func TestImageContentType(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{"PNG ผ่าน", "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR", "image/png"},
		{"JPG ผ่าน", "\xff\xd8\xff\xe0\x00\x10JFIF\x00", "image/jpeg"},
		{"WebP ผ่าน", "RIFF\x24\x00\x00\x00WEBPVP8 ", "image/webp"},
		{"GIF ไม่รับ", "GIF89a\x01\x00\x01\x00", ""},
		{"SVG ไม่รับ", `<svg xmlns="http://www.w3.org/2000/svg"></svg>`, ""},
		{"ไฟล์ข้อความ ไม่รับ", "hello", ""},
		{"ไฟล์ PDF ไม่รับ", "%PDF-1.7", ""},
		{"ไฟล์ว่าง ไม่รับ", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ImageContentType([]byte(tt.data))
			if got != tt.want {
				t.Errorf("ImageContentType() = %q, ต้องได้ %q", got, tt.want)
			}
		})
	}
}
