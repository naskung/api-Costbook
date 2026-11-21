package utils

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

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

// solidImage รูปสีเดียวขนาด width × height ไว้ใช้ใน test
func solidImage(width, height int, fill color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.NewUniform(fill), image.Point{}, draw.Src)
	return img
}

func TestSquareIcon(t *testing.T) {
	red := color.RGBA{R: 255, A: 255}
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	// รูปกว้าง 4 × 2 ใส่กรอบ 10 ไม่เว้นขอบ → ได้ 10 × 5 อยู่กลาง (แถว 0–1 และ 7–9 เป็นพื้นขาว)
	icon := SquareIcon(solidImage(4, 2, red), 10, 0)
	if icon.Bounds() != image.Rect(0, 0, 10, 10) {
		t.Fatalf("ขนาด = %v, ต้องได้ 10 × 10", icon.Bounds())
	}
	if icon.RGBAAt(5, 0) != white || icon.RGBAAt(5, 9) != white {
		t.Errorf("บน / ล่างของรูปกว้างต้องเป็นพื้นขาว")
	}
	if icon.RGBAAt(0, 4) != red || icon.RGBAAt(9, 4) != red {
		t.Errorf("กลางรูปต้องเป็นสีของรูปเต็มความกว้าง")
	}

	// เว้นขอบ 10% → ขอบซ้ายสุดเป็นพื้นขาว
	withMargin := SquareIcon(solidImage(4, 4, red), 10, 10)
	if withMargin.RGBAAt(0, 5) != white || withMargin.RGBAAt(1, 5) != red {
		t.Errorf("เว้นขอบ 10%% ต้องได้ขอบขาว 1 จุด แล้วเป็นรูป")
	}
}

func TestSquareIcon_TransparentBecomesWhite(t *testing.T) {
	icon := SquareIcon(solidImage(8, 8, color.Transparent), 4, 0)
	if icon.RGBAAt(2, 2) != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Errorf("ส่วนโปร่งใสต้องกลายเป็นสีขาว ได้ %v", icon.RGBAAt(2, 2))
	}
}

func TestFitSize(t *testing.T) {
	tests := []struct {
		name                  string
		width, height, box    int
		wantWidth, wantHeight int
	}{
		{"รูปกว้าง ด้านกว้างเท่ากรอบ", 400, 200, 100, 100, 50},
		{"รูปสูง ด้านสูงเท่ากรอบ", 200, 400, 100, 50, 100},
		{"รูปเล็กกว่ากรอบ ขยายขึ้น", 10, 10, 100, 100, 100},
		{"รูปเส้นบางมาก อย่างน้อย 1 จุด", 1000, 1, 10, 10, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			width, height := fitSize(tt.width, tt.height, tt.box)
			if width != tt.wantWidth || height != tt.wantHeight {
				t.Errorf("fitSize() = %d × %d, ต้องได้ %d × %d", width, height, tt.wantWidth, tt.wantHeight)
			}
		})
	}
}
