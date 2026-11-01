package user

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/naskung/api-Costbook/internal/features/auth"
	"github.com/naskung/api-Costbook/internal/middleware"
	"github.com/naskung/api-Costbook/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

const testSecret = "test-secret"

const (
	ownerID = "11111111-1111-1111-1111-111111111111"
	staffID = "22222222-2222-2222-2222-222222222222"
	otherID = "33333333-3333-3333-3333-333333333333"
)

// fakeRepository เก็บผู้ใช้ไว้ใน map / updateErr = จำลอง error ตอนแก้ผู้ใช้ (เช่น ErrLastOwner)
type fakeRepository struct {
	users     map[string]User
	hashes    map[string]string
	avatars   map[string]Avatar
	updateErr error
	created   *CreateRequest
}

func newFakeRepository(t *testing.T) *fakeRepository {
	hash, err := bcrypt.GenerateFromPassword([]byte("1234"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return &fakeRepository{
		users: map[string]User{
			ownerID: {ID: ownerID, Email: "owner@costbook.com", FirstName: "เจ้าของ", Role: auth.RoleOwner, Active: true},
			staffID: {ID: staffID, Email: "staff@costbook.com", FirstName: "พนักงาน", Role: auth.RoleStaff, Active: true},
		},
		hashes:  map[string]string{ownerID: string(hash), staffID: string(hash)},
		avatars: map[string]Avatar{},
	}
}

func (f *fakeRepository) find(id string) (User, error) {
	user, ok := f.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return user, nil
}

func (f *fakeRepository) List(ctx context.Context) ([]User, error) {
	return []User{f.users[ownerID], f.users[staffID]}, nil
}

func (f *fakeRepository) Find(ctx context.Context, id string) (User, error) { return f.find(id) }

func (f *fakeRepository) PasswordHash(ctx context.Context, id string) (string, error) {
	if _, err := f.find(id); err != nil {
		return "", err
	}
	return f.hashes[id], nil
}

func (f *fakeRepository) Create(ctx context.Context, req CreateRequest, passwordHash string) (User, error) {
	for _, u := range f.users {
		if u.Email == normalizeEmail(req.Email) {
			return User{}, ErrEmailTaken
		}
	}
	f.created = &req
	user := User{ID: otherID, Email: normalizeEmail(req.Email), FirstName: req.FirstName, Role: req.Role, Active: true}
	f.users[otherID], f.hashes[otherID] = user, passwordHash
	return user, nil
}

func (f *fakeRepository) Update(ctx context.Context, id string, req UpdateRequest) (User, error) {
	user, err := f.find(id)
	if err != nil || f.updateErr != nil {
		return User{}, errors.Join(err, f.updateErr)
	}
	user.FirstName, user.Role = req.FirstName, req.Role
	if req.Active != nil {
		user.Active = *req.Active
	}
	f.users[id] = user
	return user, nil
}

func (f *fakeRepository) UpdateProfile(ctx context.Context, id string, req ProfileRequest) (User, error) {
	user, err := f.find(id)
	if err != nil {
		return User{}, err
	}
	user.FirstName, user.LastName = req.FirstName, req.LastName
	f.users[id] = user
	return user, nil
}

func (f *fakeRepository) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	if _, err := f.find(id); err != nil {
		return err
	}
	f.hashes[id] = passwordHash
	return nil
}

func (f *fakeRepository) FindAvatar(ctx context.Context, id string) (Avatar, error) {
	if _, err := f.find(id); err != nil {
		return Avatar{}, err
	}
	avatar, ok := f.avatars[id]
	if !ok {
		return Avatar{}, ErrAvatarNotFound
	}
	return avatar, nil
}

func (f *fakeRepository) SaveAvatar(ctx context.Context, id string, avatar Avatar) (User, error) {
	user, err := f.find(id)
	if err != nil {
		return User{}, err
	}
	f.avatars[id] = avatar
	user.AvatarVersion = 1
	f.users[id] = user
	return user, nil
}

func (f *fakeRepository) DeleteAvatar(ctx context.Context, id string) (User, error) {
	user, err := f.find(id)
	if err != nil {
		return User{}, err
	}
	delete(f.avatars, id)
	user.AvatarVersion = 0
	f.users[id] = user
	return user, nil
}

// newTestRouter route เหมือนของจริง: Authenticated อ่าน role / สถานะจาก fakeRepository
func newTestRouter(repo *fakeRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	lookup := func(ctx context.Context, id string) (string, bool, error) {
		user, err := repo.find(id)
		if err != nil {
			return "", false, nil
		}
		return user.Role, user.Active, nil
	}
	h := NewHandler(repo)
	h.hashCost = bcrypt.MinCost
	h.RegisterRoutes(r.Group("/api/v1"), middleware.Authenticated(testSecret, lookup))
	return r
}

func send(router *gin.Engine, userID, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	token, _ := utils.CreateToken(userID, testSecret, time.Hour)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func wantError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status || !strings.Contains(rec.Body.String(), `"`+code+`"`) {
		t.Errorf("ต้องได้ %d %s ได้ %d %s", status, code, rec.Code, rec.Body.String())
	}
}

func passwordIs(repo *fakeRepository, id, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(repo.hashes[id]), []byte(password)) == nil
}

// ---------- โปรไฟล์ ----------

func TestHandler_UpdateProfile(t *testing.T) {
	repo := newFakeRepository(t)
	rec := send(newTestRouter(repo), staffID, http.MethodPut, "/api/v1/profile", `{"first_name":"สมหญิง","last_name":"ใจดี"}`)
	if rec.Code != http.StatusOK || repo.users[staffID].FirstName != "สมหญิง" || !strings.Contains(rec.Body.String(), `"role":"staff"`) {
		t.Errorf("พนักงานแก้ชื่อตัวเองได้ ได้ %d %s", rec.Code, rec.Body.String())
	}
	wantError(t, send(newTestRouter(repo), staffID, http.MethodPut, "/api/v1/profile", `{"first_name":""}`), http.StatusBadRequest, "user_name_required")
	wantError(t, send(newTestRouter(repo), staffID, http.MethodPut, "/api/v1/profile", `{`), http.StatusBadRequest, "invalid_json")
}

func TestHandler_ChangePassword(t *testing.T) {
	repo := newFakeRepository(t)
	router := newTestRouter(repo)

	wantError(t, send(router, staffID, http.MethodPut, "/api/v1/profile/password", `{"current_password":"wrong","new_password":"newpass1"}`),
		http.StatusBadRequest, "password_current_invalid")
	wantError(t, send(router, staffID, http.MethodPut, "/api/v1/profile/password", `{"current_password":"1234","new_password":"123"}`),
		http.StatusBadRequest, "password_too_short")
	if !passwordIs(repo, staffID, "1234") {
		t.Fatal("ยังไม่ผ่าน ต้องไม่เปลี่ยนรหัส")
	}

	rec := send(router, staffID, http.MethodPut, "/api/v1/profile/password", `{"current_password":"1234","new_password":"newpass1"}`)
	if rec.Code != http.StatusNoContent || !passwordIs(repo, staffID, "newpass1") || !passwordIs(repo, ownerID, "1234") {
		t.Errorf("เปลี่ยนเฉพาะรหัสของตัวเอง ได้ %d", rec.Code)
	}
}

func avatarRequest(t *testing.T, field string, data []byte) (*http.Request, string) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(field, "me.png")
	if err != nil {
		t.Fatal(err)
	}
	part.Write(data)
	writer.Close()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/profile/avatar", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, writer.FormDataContentType()
}

// pngBytes ไฟล์ PNG เล็ก ๆ (ดูแค่หัวไฟล์ว่าเป็น PNG)
var pngBytes = append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...)

func TestHandler_Avatar(t *testing.T) {
	repo := newFakeRepository(t)
	router := newTestRouter(repo)
	token, _ := utils.CreateToken(staffID, testSecret, time.Hour)

	req, _ := avatarRequest(t, "avatar", pngBytes)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"avatar_version":1`) {
		t.Fatalf("อัปโหลดรูปโปรไฟล์ได้ ได้ %d %s", rec.Code, rec.Body.String())
	}

	got := send(router, ownerID, http.MethodGet, "/api/v1/users/"+staffID+"/avatar?v=1", "")
	if got.Code != http.StatusOK || got.Header().Get("Content-Type") != "image/png" || !strings.Contains(got.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("คนอื่นที่ login ดูรูปได้ + cache นาน ได้ %d %v", got.Code, got.Header())
	}

	if rec := send(router, staffID, http.MethodDelete, "/api/v1/profile/avatar", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"avatar_version":0`) {
		t.Errorf("ลบรูปได้ ได้ %d %s", rec.Code, rec.Body.String())
	}
	wantError(t, send(router, ownerID, http.MethodGet, "/api/v1/users/"+staffID+"/avatar", ""), http.StatusNotFound, "avatar_not_found")
	wantError(t, send(router, ownerID, http.MethodGet, "/api/v1/users/"+otherID+"/avatar", ""), http.StatusNotFound, "user_not_found")
	wantError(t, send(router, ownerID, http.MethodGet, "/api/v1/users/not-uuid/avatar", ""), http.StatusNotFound, "user_not_found")
}

func TestHandler_Avatar_BadFile(t *testing.T) {
	router := newTestRouter(newFakeRepository(t))
	token, _ := utils.CreateToken(staffID, testSecret, time.Hour)
	tests := []struct {
		name  string
		field string
		data  []byte
		code  string
	}{
		{"ไม่ใช่รูป", "avatar", []byte("hello"), "avatar_type_invalid"},
		{"ชื่อช่องผิด", "image", pngBytes, "avatar_required"},
	}
	for _, tt := range tests {
		req, _ := avatarRequest(t, tt.field, tt.data)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		wantError(t, rec, http.StatusBadRequest, tt.code)
	}
}

// ---------- จัดการผู้ใช้ ----------

func TestHandler_ManageUsers_OwnerOnly(t *testing.T) {
	repo := newFakeRepository(t)
	repo.users[otherID] = User{ID: otherID, Email: "manager@costbook.com", FirstName: "ผู้จัดการ", Role: auth.RoleManager, Active: true}
	router := newTestRouter(repo)
	routes := []struct{ method, path, body string }{
		{http.MethodGet, "/api/v1/users", ""},
		{http.MethodPost, "/api/v1/users", `{"email":"x@costbook.com","first_name":"x","role":"staff","password":"123456"}`},
		{http.MethodPut, "/api/v1/users/" + staffID, `{"first_name":"x","role":"owner"}`},
		{http.MethodPut, "/api/v1/users/" + staffID + "/password", `{"new_password":"123456"}`},
	}
	for _, userID := range []string{staffID, otherID} {
		for _, route := range routes {
			wantError(t, send(router, userID, route.method, route.path, route.body), http.StatusForbidden, "forbidden")
		}
	}
	if repo.users[staffID].Role != auth.RoleStaff || repo.created != nil {
		t.Error("ไม่ใช่เจ้าของ ต้องไม่ได้แก้อะไรเลย")
	}
}

func TestHandler_List(t *testing.T) {
	rec := send(newTestRouter(newFakeRepository(t)), ownerID, http.MethodGet, "/api/v1/users", "")
	var users []User
	json.NewDecoder(rec.Body).Decode(&users)
	if rec.Code != http.StatusOK || len(users) != 2 || strings.Contains(rec.Body.String(), "password") {
		t.Errorf("เจ้าของดูรายชื่อได้ ไม่มีรหัสผ่านออกไป ได้ %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_Create(t *testing.T) {
	repo := newFakeRepository(t)
	router := newTestRouter(repo)
	rec := send(router, ownerID, http.MethodPost, "/api/v1/users", `{"email":" New@CostBook.com ","first_name":"ใหม่","role":"manager","password":"secret1"}`)
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), `"email":"new@costbook.com"`) || !passwordIs(repo, otherID, "secret1") {
		t.Errorf("เพิ่มผู้ใช้ได้ อีเมลเป็นตัวเล็ก รหัสเก็บแบบ hash ได้ %d %s", rec.Code, rec.Body.String())
	}
	wantError(t, send(router, ownerID, http.MethodPost, "/api/v1/users", `{"email":"STAFF@costbook.com","first_name":"ซ้ำ","role":"staff","password":"123456"}`),
		http.StatusConflict, "user_email_taken")
	wantError(t, send(router, ownerID, http.MethodPost, "/api/v1/users", `{"email":"x@costbook.com","first_name":"x","role":"admin","password":"123456"}`),
		http.StatusBadRequest, "user_role_invalid")
}

func TestHandler_Update(t *testing.T) {
	repo := newFakeRepository(t)
	router := newTestRouter(repo)
	rec := send(router, ownerID, http.MethodPut, "/api/v1/users/"+staffID, `{"first_name":"พนักงาน","role":"manager","active":false}`)
	if rec.Code != http.StatusOK || repo.users[staffID].Role != auth.RoleManager || repo.users[staffID].Active {
		t.Errorf("เจ้าของเปลี่ยน role + ปิดบัญชีได้ ได้ %d %s", rec.Code, rec.Body.String())
	}
	if rec := send(router, staffID, http.MethodGet, "/api/v1/users/"+staffID+"/avatar", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("บัญชีถูกปิดแล้ว token เดิมใช้ไม่ได้ทันที ได้ %d", rec.Code)
	}

	wantError(t, send(router, ownerID, http.MethodPut, "/api/v1/users/"+otherID, `{"first_name":"x","role":"staff"}`), http.StatusNotFound, "user_not_found")
	repo.updateErr = ErrLastOwner
	wantError(t, send(router, ownerID, http.MethodPut, "/api/v1/users/"+ownerID, `{"first_name":"x","role":"staff"}`), http.StatusConflict, "user_last_owner")
}

func TestHandler_ResetPassword(t *testing.T) {
	repo := newFakeRepository(t)
	router := newTestRouter(repo)
	rec := send(router, ownerID, http.MethodPut, "/api/v1/users/"+staffID+"/password", `{"new_password":"reset99"}`)
	if rec.Code != http.StatusNoContent || !passwordIs(repo, staffID, "reset99") {
		t.Errorf("เจ้าของตั้งรหัสใหม่ให้พนักงานได้ (ไม่ต้องรู้รหัสเดิม) ได้ %d", rec.Code)
	}
	wantError(t, send(router, ownerID, http.MethodPut, "/api/v1/users/"+otherID+"/password", `{"new_password":"reset99"}`), http.StatusNotFound, "user_not_found")
}
