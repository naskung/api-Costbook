package database

import (
	"context"
	"testing"
)

func TestConnect_Error(t *testing.T) {
	tests := []struct {
		name        string
		databaseURL string
	}{
		{"URL ผิดรูปแบบ ได้ error", "postgres://%%%"},
		{"ต่อ server ไม่ได้ ได้ error", "postgres://nobody@127.0.0.1:1/costbook?connect_timeout=1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := Connect(context.Background(), tt.databaseURL)
			if err == nil {
				pool.Close()
				t.Error("Connect() error = nil, ต้องได้ error")
			}
		})
	}
}
