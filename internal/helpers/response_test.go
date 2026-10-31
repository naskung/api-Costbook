package helpers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWriteError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	WriteError(c, http.StatusBadRequest, "invalid_json")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, ต้องได้ %d", rec.Code, http.StatusBadRequest)
	}
	if rec.Body.String() != `{"error":"invalid_json"}` {
		t.Errorf("body = %s, ต้องได้ %s", rec.Body.String(), `{"error":"invalid_json"}`)
	}
	if !c.IsAborted() {
		t.Error("ต้องหยุด (abort) ไม่ให้ทำ handler ถัดไป")
	}
}

func TestServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)

	ServerError(c, errors.New("password for db is 1234"))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, ต้องได้ %d", rec.Code, http.StatusInternalServerError)
	}
	if rec.Body.String() != `{"error":"server_error"}` {
		t.Errorf("body = %s, ต้องได้ %s", rec.Body.String(), `{"error":"server_error"}`)
	}
	if strings.Contains(rec.Body.String(), "1234") {
		t.Error("ห้ามส่งข้อความ error จริงให้ client")
	}
}
