package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/middleware"
	"github.com/naskung/api-Costbook/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

const testSecret = "test-secret"

// fakeRepository ฐานข้อมูลปลอม เก็บข้อมูลใน slice ใช้แทน Postgres ตอน test
type fakeRepository struct {
	users []User
	err   error // ใส่ค่านี้เมื่ออยากจำลองว่า DB พัง
}

func (f *fakeRepository) FindByEmail(ctx context.Context, email string) (User, error) {
	if f.err != nil {
		return User{}, f.err
	}
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return User{}, ErrNotFound
}

func (f *fakeRepository) FindByID(ctx context.Context, id string) (User, error) {
	if f.err != nil {
		return User{}, f.err
	}
	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return User{}, ErrNotFound
}

func (f *fakeRepository) Status(ctx context.Context, id string) (string, bool, error) {
	user, err := f.FindByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return "", false, nil
	}
	return user.Role, user.Active, err
}

// newTestRouter สร้าง Gin router สำหรับ test: route เหมือนของจริง แต่ใช้ fakeRepository แทน DB
func newTestRouter(repo Repository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler(repo, testSecret).RegisterRoutes(r.Group("/api/v1"), middleware.Auth(testSecret))
	return r
}

// newAdmin ผู้ใช้ทดสอบ รหัสผ่าน "1234" (ใช้ MinCost ให้ test เร็ว)
func newAdmin(t *testing.T) User {
	hash, err := bcrypt.GenerateFromPassword([]byte("1234"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("สร้าง hash ไม่ได้: %v", err)
	}
	return User{ID: "user-1", Email: "admin@costbook.com", FirstName: "Admin", LastName: "CostBook", Role: RoleOwner, Active: true, PasswordHash: string(hash)}
}

func postLogin(router *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestHandler_Login_Success(t *testing.T) {
	router := newTestRouter(&fakeRepository{users: []User{newAdmin(t)}})

	// อีเมลตัวพิมพ์ใหญ่ + มีช่องว่าง ก็ต้อง login ได้
	rec := postLogin(router, `{"email":"  Admin@CostBook.com ","password":"1234"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ต้องได้ %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "password") {
		t.Errorf("ห้ามส่ง password_hash ออกไป: %s", rec.Body.String())
	}

	var got LoginResponse
	json.NewDecoder(rec.Body).Decode(&got)
	if got.User.ID != "user-1" || got.User.Email != "admin@costbook.com" || got.User.FirstName != "Admin" {
		t.Errorf("user = %+v, ต้องได้ข้อมูลของ admin", got.User)
	}

	userID, err := utils.ParseToken(got.Token, testSecret)
	if err != nil || userID != "user-1" {
		t.Errorf("token ใช้ไม่ได้: userID = %q, err = %v", userID, err)
	}
}

func TestHandler_Login_Fail(t *testing.T) {
	tests := []struct {
		name       string
		repo       *fakeRepository
		body       string
		wantStatus int
		wantError  string
	}{
		{"JSON ผิดรูปแบบ ได้ 400", &fakeRepository{}, `{"email":`, http.StatusBadRequest, "invalid_json"},
		{"อีเมลว่าง ได้ 400", &fakeRepository{}, `{"email":"","password":"1234"}`, http.StatusBadRequest, "email_required"},
		{"รหัสผ่านว่าง ได้ 400", &fakeRepository{}, `{"email":"admin@costbook.com","password":""}`, http.StatusBadRequest, "password_required"},
		{"ไม่มีอีเมลนี้ ได้ 401", &fakeRepository{users: []User{newAdmin(t)}}, `{"email":"nobody@costbook.com","password":"1234"}`, http.StatusUnauthorized, "invalid_credentials"},
		{"รหัสผ่านผิด ได้ 401", &fakeRepository{users: []User{newAdmin(t)}}, `{"email":"admin@costbook.com","password":"wrong"}`, http.StatusUnauthorized, "invalid_credentials"},
		{"DB พัง ได้ 500", &fakeRepository{err: errors.New("db down")}, `{"email":"admin@costbook.com","password":"1234"}`, http.StatusInternalServerError, "server_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := postLogin(newTestRouter(tt.repo), tt.body)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, ต้องได้ %d", rec.Code, tt.wantStatus)
			}
			wantBody := `{"error":"` + tt.wantError + `"}`
			if rec.Body.String() != wantBody {
				t.Errorf("body = %s, ต้องได้ %s", rec.Body.String(), wantBody)
			}
		})
	}
}

func getMe(router *gin.Engine, userID string) *httptest.ResponseRecorder {
	token, _ := utils.CreateToken(userID, testSecret, time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestHandler_Me(t *testing.T) {
	router := newTestRouter(&fakeRepository{users: []User{newAdmin(t)}})

	rec := getMe(router, "user-1")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, ต้องได้ %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var got User
	json.NewDecoder(rec.Body).Decode(&got)
	if got.ID != "user-1" || got.Email != "admin@costbook.com" || got.LastName != "CostBook" {
		t.Errorf("user = %+v, ต้องได้ข้อมูลของ admin", got)
	}
}

func TestHandler_Me_Fail(t *testing.T) {
	tests := []struct {
		name       string
		repo       *fakeRepository
		wantStatus int
		wantError  string
	}{
		{"ผู้ใช้ถูกลบไปแล้ว ได้ 401", &fakeRepository{}, http.StatusUnauthorized, "unauthorized"},
		{"DB พัง ได้ 500", &fakeRepository{err: errors.New("db down")}, http.StatusInternalServerError, "server_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := getMe(newTestRouter(tt.repo), "user-1")

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, ต้องได้ %d", rec.Code, tt.wantStatus)
			}
			wantBody := `{"error":"` + tt.wantError + `"}`
			if rec.Body.String() != wantBody {
				t.Errorf("body = %s, ต้องได้ %s", rec.Body.String(), wantBody)
			}
		})
	}
}

func TestHandler_Me_NoToken(t *testing.T) {
	router := newTestRouter(&fakeRepository{users: []User{newAdmin(t)}})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, ต้องได้ %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestHandler_Login_DisabledUser(t *testing.T) {
	disabled := newAdmin(t)
	disabled.Active = false
	rec := postLogin(newTestRouter(&fakeRepository{users: []User{disabled}}), `{"email":"admin@costbook.com","password":"1234"}`)
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "user_disabled") {
		t.Errorf("บัญชีถูกปิด ต้องได้ 403 user_disabled ได้ %d %s", rec.Code, rec.Body.String())
	}
	wrong := postLogin(newTestRouter(&fakeRepository{users: []User{disabled}}), `{"email":"admin@costbook.com","password":"wrong"}`)
	if wrong.Code != http.StatusUnauthorized {
		t.Errorf("รหัสผิด ไม่ต้องบอกว่าบัญชีถูกปิด ได้ %d", wrong.Code)
	}
}

func TestHandler_Login_ReturnsRole(t *testing.T) {
	rec := postLogin(newTestRouter(&fakeRepository{users: []User{newAdmin(t)}}), `{"email":"admin@costbook.com","password":"1234"}`)
	if !strings.Contains(rec.Body.String(), `"role":"owner"`) || !strings.Contains(rec.Body.String(), `"avatar_version":0`) {
		t.Errorf("ต้องส่ง role + avatar_version ให้ Flutter ได้ %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `"active"`) {
		t.Errorf("ไม่ต้องส่ง active ออกไป ได้ %s", rec.Body.String())
	}
}
