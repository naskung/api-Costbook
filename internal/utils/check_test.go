package utils

import "testing"

func TestIsTaxID(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"ตัวเลข 13 หลัก ผ่าน", "0105555555555", true},
		{"12 หลัก ไม่ผ่าน", "010555555555", false},
		{"14 หลัก ไม่ผ่าน", "01055555555555", false},
		{"มีตัวอักษร ไม่ผ่าน", "010555555555A", false},
		{"มีขีด ไม่ผ่าน", "0-1055-55555-55", false},
		{"ตัวเลขไทย ไม่ผ่าน", "๐๑๐๕๕๕๕๕๕๕๕๕๕", false},
		{"ค่าว่าง ไม่ผ่าน", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTaxID(tt.value)
			if got != tt.want {
				t.Errorf("IsTaxID(%q) = %v, ต้องได้ %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestIsUUID(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"UUID ตัวเล็ก ผ่าน", "a00d265d-5466-4cba-aba7-005953ae30fc", true},
		{"UUID ตัวใหญ่ ผ่าน", "A00D265D-5466-4CBA-ABA7-005953AE30FC", true},
		{"ตัวเลขธรรมดา ไม่ผ่าน", "123", false},
		{"ไม่มีขีด ไม่ผ่าน", "a00d265d54664cbaaba7005953ae30fc0000", false},
		{"มีตัวอักษรที่ไม่ใช่ hex ไม่ผ่าน", "z00d265d-5466-4cba-aba7-005953ae30fc", false},
		{"ค่าว่าง ไม่ผ่าน", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUUID(tt.value)
			if got != tt.want {
				t.Errorf("IsUUID(%q) = %v, ต้องได้ %v", tt.value, got, tt.want)
			}
		})
	}
}
