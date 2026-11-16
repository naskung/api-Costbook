package utils

import (
	"testing"
	"time"
)

func TestIsDate(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"2026-10-31 เป็นวันที่", "2026-10-31", true},
		{"2028-02-29 ปีอธิกสุรทิน เป็นวันที่", "2028-02-29", true},
		{"2026-02-30 ไม่มีวันนี้จริง", "2026-02-30", false},
		{"31/10/2026 รูปแบบผิด", "31/10/2026", false},
		{"ค่าว่าง ไม่ใช่วันที่", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDate(tt.value)
			if got != tt.want {
				t.Errorf("IsDate(%q) = %v, ต้องได้ %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestNewDateRange(t *testing.T) {
	tests := []struct {
		name   string
		from   string
		to     string
		wantOK bool
	}{
		{"1–16 พ.ย. ผ่าน", "2026-11-01", "2026-11-16", true},
		{"วันเดียว (วันนี้ถึงวันนี้) ผ่าน", "2026-11-16", "2026-11-16", true},
		{"ทั้งปี 366 วัน (ปีอธิกสุรทิน) ผ่าน", "2028-01-01", "2028-12-31", true},
		{"367 วัน ยาวเกิน", "2026-01-01", "2027-01-02", false},
		{"from อยู่หลัง to", "2026-11-16", "2026-11-01", false},
		{"from รูปแบบผิด", "16/11/2026", "2026-11-16", false},
		{"to ไม่มีวันนี้จริง", "2026-02-01", "2026-02-30", false},
		{"ค่าว่าง", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := NewDateRange(tt.from, tt.to)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, ต้องได้ %v", ok, tt.wantOK)
			}
			if ok && (got.From != tt.from || got.To != tt.to) {
				t.Errorf("ได้ %+v, ต้องได้ %s ถึง %s", got, tt.from, tt.to)
			}
		})
	}
}

func TestMonthRange(t *testing.T) {
	tests := []struct {
		day  time.Time
		want DateRange
	}{
		{time.Date(2026, 11, 16, 15, 30, 0, 0, time.Local), DateRange{"2026-11-01", "2026-11-30"}},
		{time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), DateRange{"2026-12-01", "2026-12-31"}},
		{time.Date(2028, 2, 10, 0, 0, 0, 0, time.UTC), DateRange{"2028-02-01", "2028-02-29"}},
	}
	for _, tt := range tests {
		if got := MonthRange(tt.day); got != tt.want {
			t.Errorf("MonthRange(%s) = %+v, ต้องได้ %+v", tt.day.Format(DateLayout), got, tt.want)
		}
	}
}

func TestPreviousRange(t *testing.T) {
	tests := []struct {
		name  string
		input DateRange
		want  DateRange
	}{
		{"ทั้งเดือน พ.ย. → ต.ค. ทั้งเดือน", DateRange{"2026-11-01", "2026-11-30"}, DateRange{"2026-10-01", "2026-10-31"}},
		{"ทั้งเดือน ม.ค. → ธ.ค. ปีก่อน", DateRange{"2026-01-01", "2026-01-31"}, DateRange{"2025-12-01", "2025-12-31"}},
		{"ทั้งเดือน มี.ค. → ก.พ. (28 วัน)", DateRange{"2026-03-01", "2026-03-31"}, DateRange{"2026-02-01", "2026-02-28"}},
		{"1–15 พ.ย. → 17–31 ต.ค.", DateRange{"2026-11-01", "2026-11-15"}, DateRange{"2026-10-17", "2026-10-31"}},
		{"วันเดียว → เมื่อวาน", DateRange{"2026-11-16", "2026-11-16"}, DateRange{"2026-11-15", "2026-11-15"}},
		{"7 วัน 10–16 พ.ย. → 3–9 พ.ย.", DateRange{"2026-11-10", "2026-11-16"}, DateRange{"2026-11-03", "2026-11-09"}},
		{"รูปแบบผิด คืนค่าเดิม", DateRange{"abc", "2026-11-16"}, DateRange{"abc", "2026-11-16"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PreviousRange(tt.input); got != tt.want {
				t.Errorf("ได้ %+v, ต้องได้ %+v", got, tt.want)
			}
		})
	}
}
