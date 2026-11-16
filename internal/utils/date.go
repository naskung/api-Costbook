package utils

import "time"

// DateLayout รูปแบบวันที่ที่รับ / ส่งกับ Flutter เช่น "2026-10-31"
const DateLayout = "2006-01-02"

// MaxRangeDays เลือกช่วงวันที่ได้ยาวสุดกี่วัน (1 ปี เผื่อปีอธิกสุรทิน)
const MaxRangeDays = 366

// IsDate เช็คว่าเป็นวันที่ที่มีอยู่จริงในรูปแบบ "2026-10-31" ไหม
func IsDate(value string) bool {
	_, err := time.Parse(DateLayout, value)
	return err == nil
}

// DateRange ช่วงวันที่ From ถึง To (นับรวมทั้ง 2 วัน) รูปแบบ "2026-11-01"
type DateRange struct {
	From string
	To   string
}

// NewDateRange ตรวจช่วงวันที่ — รูปแบบผิด / from อยู่หลัง to / ยาวเกิน 366 วัน → ok = false
func NewDateRange(from, to string) (DateRange, bool) {
	start, err := time.Parse(DateLayout, from)
	if err != nil {
		return DateRange{}, false
	}
	end, err := time.Parse(DateLayout, to)
	if err != nil || end.Before(start) {
		return DateRange{}, false
	}
	if daysBetween(start, end)+1 > MaxRangeDays {
		return DateRange{}, false
	}
	return DateRange{From: from, To: to}, true
}

// MonthRange ทั้งเดือนของวันที่ที่ให้มา เช่น 16 พ.ย. 2026 → 2026-11-01 ถึง 2026-11-30
func MonthRange(day time.Time) DateRange {
	start := time.Date(day.Year(), day.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, -1)
	return DateRange{From: start.Format(DateLayout), To: end.Format(DateLayout)}
}

// PreviousRange ช่วงก่อนหน้า ไว้เทียบว่ายอดขึ้นหรือลง
//   - เลือกทั้งเดือนพอดี → เดือนก่อนทั้งเดือน เช่น 1–30 พ.ย. → 1–31 ต.ค.
//   - นอกนั้น → ช่วงที่ยาวเท่ากัน ก่อนหน้าติดกัน เช่น 1–15 พ.ย. → 17–31 ต.ค.
func PreviousRange(r DateRange) DateRange {
	start, err := time.Parse(DateLayout, r.From)
	if err != nil {
		return r
	}
	end, err := time.Parse(DateLayout, r.To)
	if err != nil {
		return r
	}
	if MonthRange(start) == r {
		return MonthRange(start.AddDate(0, -1, 0))
	}
	previousEnd := start.AddDate(0, 0, -1)
	previousStart := previousEnd.AddDate(0, 0, -daysBetween(start, end))
	return DateRange{From: previousStart.Format(DateLayout), To: previousEnd.Format(DateLayout)}
}

// daysBetween ห่างกันกี่วัน เช่น 1 พ.ย. ถึง 15 พ.ย. = 14
func daysBetween(start, end time.Time) int {
	return int(end.Sub(start).Hours() / 24)
}
