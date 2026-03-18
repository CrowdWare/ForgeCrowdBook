package auth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestCreateAndGetSession(t *testing.T) {
	secret := "test-secret"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	CreateSession(rec, "user@example.com", secret)
	resp := rec.Result()
	defer resp.Body.Close()

	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie")
	}
	req.AddCookie(cookies[0])

	email, ok := GetSession(req, secret)
	if !ok {
		t.Fatal("expected valid session")
	}
	if email != "user@example.com" {
		t.Fatalf("unexpected email %q", email)
	}
}

func TestGetSessionRejectsTamperedCookie(t *testing.T) {
	secret := "test-secret"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	CreateSession(rec, "user@example.com", secret)
	resp := rec.Result()
	defer resp.Body.Close()

	cookie := resp.Cookies()[0]
	cookie.Value = cookie.Value[:len(cookie.Value)-1] + "A"
	req.AddCookie(cookie)

	if _, ok := GetSession(req, secret); ok {
		t.Fatal("expected tampered cookie to be rejected")
	}
}

func TestGetSessionRejectsExpiredCookie(t *testing.T) {
	secret := "test-secret"
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	payload := "user@example.com|" + strconv.FormatInt(time.Now().Add(-time.Hour).Unix(), 10)
	value := payload + "|" + sign(payload, secret)
	encoded := base64.RawURLEncoding.EncodeToString([]byte(value))
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: encoded})

	if _, ok := GetSession(req, secret); ok {
		t.Fatal("expected expired cookie to be rejected")
	}
}

func TestRequireAuthRedirectsWithoutSession(t *testing.T) {
	handler := RequireAuth("test-secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/login" {
		t.Fatalf("expected redirect to /login, got %q", got)
	}
}

func TestClearSession(t *testing.T) {
	rec := httptest.NewRecorder()
	ClearSession(rec)
	resp := rec.Result()
	defer resp.Body.Close()

	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected cookie to be set")
	}
	if cookies[0].MaxAge >= 0 {
		t.Fatalf("expected negative MaxAge, got %d", cookies[0].MaxAge)
	}
}

func TestRequireAdminForbiddenForNonAdmin(t *testing.T) {
	secret := "test-secret"
	recLogin := httptest.NewRecorder()
	CreateSession(recLogin, "user@example.com", secret)
	sessionCookie := recLogin.Result().Cookies()[0]

	handler := RequireAdmin(secret, "admin@example.com", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestRequireAdminAllowsAdmin(t *testing.T) {
	secret := "test-secret"
	recLogin := httptest.NewRecorder()
	CreateSession(recLogin, "admin@example.com", secret)
	sessionCookie := recLogin.Result().Cookies()[0]

	handler := RequireAdmin(secret, "admin@example.com", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsAdminFromContext(r.Context()) {
			t.Fatal("expected admin in request context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}
