package utils

import (
	"image"
	"image/color"
	"image/draw"
	"net/http"
)

// ImageContentType ดูจากข้างในไฟล์ (ไม่เชื่อนามสกุลไฟล์) ว่าเป็นรูปชนิดไหน
// รับแค่ PNG / JPG / WebP ถ้าเป็นอย่างอื่นคืน ""
func ImageContentType(data []byte) string {
	contentType := http.DetectContentType(data)
	switch contentType {
	case "image/png", "image/jpeg", "image/webp":
		return contentType
	}
	return ""
}

// SquareIcon วาดรูปลงสี่เหลี่ยมจัตุรัส size × size พื้นขาว (ไว้ทำไอคอนแอป / favicon)
// ย่อหรือขยายให้พอดีกรอบโดยไม่บิดสัดส่วน เว้นขอบ marginPercent % รอบด้าน (0 = เต็มกรอบ) / ส่วนที่โปร่งใสกลายเป็นสีขาว
func SquareIcon(src image.Image, size, marginPercent int) *image.RGBA {
	icon := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(icon, icon.Bounds(), image.White, image.Point{}, draw.Src)

	box := size - 2*size*marginPercent/100
	width, height := fitSize(src.Bounds().Dx(), src.Bounds().Dy(), box)
	left := (size - width) / 2
	top := (size - height) / 2
	scaled := scaleImage(src, width, height)
	draw.Draw(icon, image.Rect(left, top, left+width, top+height), scaled, image.Point{}, draw.Over)
	return icon
}

// fitSize ขนาดใหม่ที่ด้านยาวสุดเท่ากับ box คงสัดส่วนเดิม (อย่างน้อย 1 จุด) เช่น 400 × 200 ใส่กรอบ 100 → 100 × 50
func fitSize(width, height, box int) (int, int) {
	if width >= height {
		return box, max(1, height*box/width)
	}
	return max(1, width*box/height), box
}

// scaleImage ย่อ / ขยายรูปเป็น width × height
// จุดใหม่แต่ละจุด = สีเฉลี่ยของจุดในรูปเดิมที่อยู่ในช่องนั้น → ย่อรูปใหญ่แล้วไม่แตก
func scaleImage(src image.Image, width, height int) *image.RGBA {
	bounds := src.Bounds()
	scaled := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		y0 := bounds.Min.Y + y*bounds.Dy()/height
		y1 := max(y0+1, bounds.Min.Y+(y+1)*bounds.Dy()/height)
		for x := 0; x < width; x++ {
			x0 := bounds.Min.X + x*bounds.Dx()/width
			x1 := max(x0+1, bounds.Min.X+(x+1)*bounds.Dx()/width)
			scaled.Set(x, y, averageColor(src, image.Rect(x0, y0, x1, y1)))
		}
	}
	return scaled
}

// averageColor สีเฉลี่ยของทุกจุดในกรอบ (รวมความโปร่งใสด้วย)
func averageColor(src image.Image, area image.Rectangle) color.RGBA64 {
	var r, g, b, a, count uint64
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			cr, cg, cb, ca := src.At(x, y).RGBA()
			r, g, b, a = r+uint64(cr), g+uint64(cg), b+uint64(cb), a+uint64(ca)
			count++
		}
	}
	return color.RGBA64{R: uint16(r / count), G: uint16(g / count), B: uint16(b / count), A: uint16(a / count)}
}
