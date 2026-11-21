package company

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/jpeg" // ให้ image.Decode อ่านโลโก้ JPG ได้
	"image/png"
	"strconv"

	"github.com/naskung/api-Costbook/internal/utils"
)

// defaultIcon ไอคอนเริ่มต้นของ CostBook (พื้นส้ม หนังสือสีขาว 512 × 512 แบบเดียวกับโลโก้เริ่มต้นในแอป)
// ใช้ตอนร้านยังไม่มีโลโก้ หรือโลโก้เป็นชนิดที่อ่านไม่ได้ (WebP)
//
//go:embed default_icon.png
var defaultIcon []byte

// iconMarginPercent ขอบรอบโลโก้ร้านในไอคอน — มือถือบางรุ่นตัดไอคอนเป็นวงกลม ถ้าชิดขอบมุมโลโก้จะหาย
const iconMarginPercent = 10

// iconSize อ่านขนาดไอคอนที่ขอ: 32 / 64 = favicon, 180 = iPhone / iPad, 192 / 512 = Android / คอม (ไฟล์ manifest)
func iconSize(text string) (int, bool) {
	size, err := strconv.Atoi(text)
	if err != nil {
		return 0, false
	}
	switch size {
	case 32, 64, 180, 192, 512:
		return size, true
	}
	return 0, false
}

// iconPNG ไอคอนแอปขนาด size × size เป็นไฟล์ PNG
// มีโลโก้ร้าน (PNG / JPG) = โลโก้วางกลางพื้นขาว / ไม่มี หรืออ่านไม่ได้ = ไอคอนเริ่มต้นของ CostBook
func iconPNG(logo []byte, size int) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(logo))
	margin := iconMarginPercent
	if err != nil {
		src, err = png.Decode(bytes.NewReader(defaultIcon))
		margin = 0
	}
	if err != nil {
		return nil, err
	}

	var file bytes.Buffer
	err = png.Encode(&file, utils.SquareIcon(src, size, margin))
	return file.Bytes(), err
}
