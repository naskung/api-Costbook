package barcode

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newFakeOpenFoodFacts server ปลอมแทน Open Food Facts ตอบ status + body ที่กำหนด แล้วจำ request ที่ได้รับไว้
func newFakeOpenFoodFacts(t *testing.T, status int, body string, gotRequest **http.Request) *openFoodFacts {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if gotRequest != nil {
			*gotRequest = r
		}
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	client := NewOpenFoodFacts().(*openFoodFacts)
	client.baseURL = server.URL
	return client
}

func TestOpenFoodFacts_Find(t *testing.T) {
	var gotRequest *http.Request
	body := `{"code":"8850124003850","status":1,"product":{"product_name":"Nestle Pure Life","product_name_th":"","brands":"Nestlé,Nestle","quantity":"600 mL"}}`
	client := newFakeOpenFoodFacts(t, http.StatusOK, body, &gotRequest)

	got, err := client.Find(context.Background(), "8850124003850")
	if err != nil {
		t.Fatalf("ต้องไม่มี error ได้ %v", err)
	}
	want := Product{Code: "8850124003850", Name: "Nestle Pure Life", Brand: "Nestlé", Quantity: "600 mL"}
	if got != want {
		t.Errorf("ได้ %+v, ต้องได้ %+v", got, want)
	}
	if gotRequest.URL.Path != "/api/v2/product/8850124003850" {
		t.Errorf("path = %q, ต้องเป็น /api/v2/product/8850124003850", gotRequest.URL.Path)
	}
	if gotRequest.Header.Get("User-Agent") != openFoodFactsUserAgent {
		t.Errorf("ต้องส่ง User-Agent เป็นชื่อแอป ได้ %q", gotRequest.Header.Get("User-Agent"))
	}
}

func TestOpenFoodFacts_Find_PrefersThaiName(t *testing.T) {
	body := `{"status":1,"product":{"product_name_th":"น้ำดื่มเนสท์เล่","product_name":"Nestle Pure Life"}}`
	client := newFakeOpenFoodFacts(t, http.StatusOK, body, nil)

	got, _ := client.Find(context.Background(), "8850124003850")
	if got.Name != "น้ำดื่มเนสท์เล่" {
		t.Errorf("มีชื่อไทย ต้องใช้ชื่อไทย ได้ %q", got.Name)
	}
}

func TestOpenFoodFacts_Find_NotFound(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"ตอบ 404 (ไม่มีสินค้านี้)", http.StatusNotFound, `{"status":0,"status_verbose":"product not found"}`},
		{"ตอบ 200 แต่ status 0", http.StatusOK, `{"status":0}`},
		{"มีสินค้าแต่ไม่มีชื่อเลย", http.StatusOK, `{"status":1,"product":{"brands":"Nestlé"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newFakeOpenFoodFacts(t, tt.status, tt.body, nil)
			_, err := client.Find(context.Background(), "8850000000001")
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("ต้องได้ ErrNotFound ได้ %v", err)
			}
		})
	}
}

func TestOpenFoodFacts_Find_Errors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"Open Food Facts ล่ม (500)", http.StatusInternalServerError, `oops`},
		{"ถูกจำกัดจำนวนครั้ง (429)", http.StatusTooManyRequests, ``},
		{"ตอบไม่ใช่ JSON", http.StatusOK, `<html>`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newFakeOpenFoodFacts(t, tt.status, tt.body, nil)
			_, err := client.Find(context.Background(), "8850124003850")
			if err == nil || errors.Is(err, ErrNotFound) {
				t.Errorf("ต้องได้ error อื่นที่ไม่ใช่ ErrNotFound ได้ %v", err)
			}
		})
	}
}
