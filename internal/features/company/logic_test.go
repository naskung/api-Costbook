package company

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// pngLogo โลโก้ PNG สีเดียว (ใช้แทนโลโก้จริงใน test)
func pngLogo(t *testing.T, width, height int, fill color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, fill)
		}
	}
	var file bytes.Buffer
	if err := png.Encode(&file, img); err != nil {
		t.Fatal(err)
	}
	return file.Bytes()
}

// decodeIcon อ่านไอคอนที่ได้กลับมาเป็นรูป
func decodeIcon(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ไอคอนต้องเป็นไฟล์ PNG ที่อ่านได้: %v", err)
	}
	return img
}

func isOrange(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return r>>8 > 240 && g>>8 > 100 && g>>8 < 130 && b>>8 < 20
}

func TestIconSize(t *testing.T) {
	for _, text := range []string{"32", "64", "180", "192", "512"} {
		if _, ok := iconSize(text); !ok {
			t.Errorf("ขนาด %s ต้องใช้ได้", text)
		}
	}
	for _, text := range []string{"", "0", "100", "1024", "abc", "-192"} {
		if _, ok := iconSize(text); ok {
			t.Errorf("ขนาด %q ต้องใช้ไม่ได้", text)
		}
	}
}

func TestIconPNG_ShopLogo(t *testing.T) {
	blue := color.RGBA{B: 255, A: 255}
	icon := decodeIcon(t, mustIcon(t, pngLogo(t, 300, 300, blue), 192))

	if icon.Bounds() != image.Rect(0, 0, 192, 192) {
		t.Fatalf("ขนาด = %v, ต้องได้ 192 × 192", icon.Bounds())
	}
	if r, g, b, _ := icon.At(96, 96).RGBA(); r != 0 || g != 0 || b>>8 != 255 {
		t.Errorf("กลางไอคอนต้องเป็นโลโก้ร้าน (สีน้ำเงิน)")
	}
	if r, g, b, _ := icon.At(2, 2).RGBA(); r>>8 != 255 || g>>8 != 255 || b>>8 != 255 {
		t.Errorf("มุมไอคอนต้องเป็นขอบขาว (ไม่ชิดขอบ)")
	}
}

func TestIconPNG_DefaultIcon(t *testing.T) {
	tests := []struct {
		name string
		logo []byte
	}{
		{"ยังไม่มีโลโก้", nil},
		{"โลโก้ WebP (อ่านไม่ได้) ใช้ไอคอนเริ่มต้นแทน", []byte("RIFF\x24\x00\x00\x00WEBPVP8 ")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			icon := decodeIcon(t, mustIcon(t, tt.logo, 64))
			if icon.Bounds() != image.Rect(0, 0, 64, 64) {
				t.Fatalf("ขนาด = %v, ต้องได้ 64 × 64", icon.Bounds())
			}
			if !isOrange(icon.At(1, 1)) {
				t.Errorf("ไอคอนเริ่มต้นต้องเป็นพื้นส้มเต็มกรอบ ได้ %v", icon.At(1, 1))
			}
		})
	}
}

func mustIcon(t *testing.T, logo []byte, size int) []byte {
	t.Helper()
	icon, err := iconPNG(logo, size)
	if err != nil {
		t.Fatalf("iconPNG() error = %v", err)
	}
	return icon
}
