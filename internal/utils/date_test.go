package utils

import "testing"

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
