package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	authpkg "codeberg.org/crowdware/forgecrowdbook/internal/auth"
	"codeberg.org/crowdware/forgecrowdbook/internal/config"
	"codeberg.org/crowdware/forgecrowdbook/internal/db"
	"codeberg.org/crowdware/forgecrowdbook/internal/i18n"
	"codeberg.org/crowdware/forgecrowdbook/internal/model"
)

type sentMail struct {
	to      string
	subject string
	body    string
}

type fakeMailer struct {
	sent []sentMail
}

func (m *fakeMailer) SendMagicLink(to, link, lang string, bundle *i18n.Bundle) error {
	m.sent = append(m.sent, sentMail{
		to:      to,
		subject: bundle.T(lang, "email_magic_link_subject"),
		body:    link,
	})
	return nil
}

func TestLoginAndRegisterUseSameConfirmationMessage(t *testing.T) {
	h, mailer, database := newAuthHandlerForTest(t)

	if _, err := model.CreateUser(database, "known@example.com", "Known"); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	loginBody := doFormPost(t, h.Login, "/login", map[string]string{
		"email": "known@example.com",
	})
	registerBody := doFormPost(t, h.Register, "/register", map[string]string{
		"display_name": "New User",
		"email":        "new@example.com",
	})

	if loginBody != registerBody {
		t.Fatalf("expected identical confirmation responses, got\nlogin: %q\nregister: %q", loginBody, registerBody)
	}
	if !strings.Contains(loginBody, "Check your email") {
		t.Fatalf("expected confirmation text, got %q", loginBody)
	}
	if len(mailer.sent) != 2 {
		t.Fatalf("expected 2 emails sent, got %d", len(mailer.sent))
	}
}

func TestLoginUnknownEmailNoEnumeration(t *testing.T) {
	h, mailer, _ := newAuthHandlerForTest(t)

	body := doFormPost(t, h.Login, "/login", map[string]string{
		"email": "missing@example.com",
	})

	if !strings.Contains(body, "Check your email") {
		t.Fatalf("expected generic confirmation message, got %q", body)
	}
	if len(mailer.sent) != 0 {
		t.Fatalf("expected no email for unknown account, got %d", len(mailer.sent))
	}
}

func TestLoginRateLimitFriendlyError(t *testing.T) {
	h, _, database := newAuthHandlerForTest(t)

	if _, err := model.CreateUser(database, "limited@example.com", "Limited"); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		req := formRequest("/login", map[string]string{"email": "limited@example.com"})
		h.Login(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for attempt %d, got %d", i+1, rec.Code)
		}
	}

	rec := httptest.NewRecorder()
	req := formRequest("/login", map[string]string{"email": "limited@example.com"})
	h.Login(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on fourth request, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Too many login links requested") {
		t.Fatalf("expected friendly rate-limit message, got %q", rec.Body.String())
	}
}

func TestAuthInvalidTokenFriendlyError(t *testing.T) {
	h, _, _ := newAuthHandlerForTest(t)

	req := httptest.NewRequest(http.MethodGet, "/auth?token=bad", nil)
	rec := httptest.NewRecorder()
	h.Auth(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "invalid or has expired") {
		t.Fatalf("expected friendly invalid token message, got %q", rec.Body.String())
	}
}

func TestAuthValidTokenCreatesSessionAndRedirects(t *testing.T) {
	h, _, database := newAuthHandlerForTest(t)

	if _, err := model.CreateUser(database, "valid@example.com", "Valid"); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	token, err := authpkg.CreateMagicToken(database, "valid@example.com")
	if err != nil {
		t.Fatalf("CreateMagicToken failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/auth?token="+token, nil)
	rec := httptest.NewRecorder()
	h.Auth(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/dashboard" {
		t.Fatalf("expected redirect to /dashboard, got %q", got)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie")
	}
}

func TestAuthBannedUserCannotLogin(t *testing.T) {
	h, _, database := newAuthHandlerForTest(t)

	user, err := model.CreateUser(database, "banned@example.com", "Banned")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if err := model.BanUser(database, user.ID); err != nil {
		t.Fatalf("BanUser failed: %v", err)
	}

	token, err := authpkg.CreateMagicToken(database, "banned@example.com")
	if err != nil {
		t.Fatalf("CreateMagicToken failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/auth?token="+token, nil)
	rec := httptest.NewRecorder()
	h.Auth(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "" {
		t.Fatalf("did not expect redirect, got %q", got)
	}
}

func newAuthHandlerForTest(t *testing.T) (*AuthHandler, *fakeMailer, *sql.DB) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "handler.db")
	database, err := db.Open(path)
	if err != nil {
		t.Fatalf("db.Open failed: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	bundle, err := i18n.Load(filepath.Join("..", "..", "i18n"))
	if err != nil {
		t.Fatalf("i18n.Load failed: %v", err)
	}

	cfg := &config.Config{
		BaseURL:       "http://example.test",
		SessionSecret: "test-secret",
	}
	mailer := &fakeMailer{}
	return NewAuthHandler(database, cfg, bundle, mailer), mailer, database
}

func doFormPost(t *testing.T, fn http.HandlerFunc, path string, values map[string]string) string {
	t.Helper()
	rec := httptest.NewRecorder()
	req := formRequest(path, values)
	fn(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %q", rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}

func formRequest(path string, values map[string]string) *http.Request {
	form := url.Values{}
	for k, v := range values {
		form.Set(k, v)
	}
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}
