package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"codeberg.org/crowdware/forgecrowdbook/internal/auth"
	"codeberg.org/crowdware/forgecrowdbook/internal/config"
	"codeberg.org/crowdware/forgecrowdbook/internal/db"
	"codeberg.org/crowdware/forgecrowdbook/internal/i18n"
	"codeberg.org/crowdware/forgecrowdbook/internal/model"
)

type testMilestoneMailer struct {
	calls int
}

func (m *testMilestoneMailer) SendLikeMilestone(to string, count int, title, lang string, bundle *i18n.Bundle) error {
	m.calls++
	return nil
}

func TestRequireAuthRedirectsWhenMissingSession(t *testing.T) {
	cfg := &config.Config{SessionSecret: "secret"}

	protected := RequireAuth(cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	if rec.Header().Get("Location") != "/login" {
		t.Fatalf("expected /login redirect, got %q", rec.Header().Get("Location"))
	}
}

func TestRequireAdminReturnsForbiddenForNonAdmin(t *testing.T) {
	cfg := &config.Config{SessionSecret: "secret", AdminEmail: "admin@example.com"}

	protected := RequireAdmin(cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	auth.CreateSession(rec, "user@example.com", cfg.SessionSecret)
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	req.AddCookie(rec.Result().Cookies()[0])
	rec = httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestSetLangSetsSessionCookieAndRedirects(t *testing.T) {
	form := url.Values{}
	form.Set("lang", "de")
	req := httptest.NewRequest(http.MethodPost, "/lang", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "/books")

	rec := httptest.NewRecorder()
	SetLang(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	if rec.Header().Get("Location") != "/books" {
		t.Fatalf("expected referer redirect, got %q", rec.Header().Get("Location"))
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %d", len(cookies))
	}
	if cookies[0].Name != "lang" || cookies[0].Value != "de" {
		t.Fatalf("unexpected cookie %+v", cookies[0])
	}
	if cookies[0].MaxAge != 0 {
		t.Fatalf("expected session cookie max-age 0, got %d", cookies[0].MaxAge)
	}
	if cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected SameSite Lax, got %v", cookies[0].SameSite)
	}
}

func TestLogoutClearsSessionAndActiveBookCookie(t *testing.T) {
	database, cfg, bundle := newPhase4TestEnv(t)
	home := NewHomeHandler(database, cfg, bundle)

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	rec := httptest.NewRecorder()
	home.Logout(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	if rec.Header().Get("Location") != "/" {
		t.Fatalf("expected redirect to /, got %q", rec.Header().Get("Location"))
	}

	cookies := rec.Result().Cookies()
	if len(cookies) < 2 {
		t.Fatalf("expected at least 2 cookies, got %d", len(cookies))
	}
	var seenSession, seenBook bool
	for _, c := range cookies {
		if c.Name == "session" && c.MaxAge == -1 {
			seenSession = true
		}
		if c.Name == activeBookCookieName && c.MaxAge == -1 {
			seenBook = true
		}
	}
	if !seenSession || !seenBook {
		t.Fatalf("expected cleared session and active book cookies, got %+v", cookies)
	}
}

func TestDashboardChaptersRedirectsWithoutActiveBook(t *testing.T) {
	database, cfg, bundle := newPhase4TestEnv(t)
	user, err := model.CreateUser(database, "user1@example.com", "User One")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	_ = user
	handler := NewChaptersHandler(database, cfg, bundle, nil)

	req := httptest.NewRequest(http.MethodGet, "/dashboard/chapters", nil)
	withSessionCookie(t, req, "user1@example.com", cfg.SessionSecret)
	rec := httptest.NewRecorder()
	handler.List(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	if rec.Header().Get("Location") != "/dashboard" {
		t.Fatalf("expected /dashboard redirect, got %q", rec.Header().Get("Location"))
	}
}

func TestLikeAPIDeduplicatesAndReturnsJSON(t *testing.T) {
	database, cfg, bundle := newPhase4TestEnv(t)
	mailer := &testMilestoneMailer{}
	api := NewAPIHandler(database, cfg, bundle, mailer)

	user, err := model.CreateUser(database, "author@example.com", "Author")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	book, err := model.GetBookBySlug(database, "choose-your-incarnation")
	if err != nil || book == nil {
		t.Fatalf("GetBookBySlug failed: %v", err)
	}
	chapter, err := model.CreateChapter(database, book.ID, user.ID, "Like Test", "like-test", "Path", "https://example.com/chapter.md")
	if err != nil {
		t.Fatalf("CreateChapter failed: %v", err)
	}

	firstReq := httptest.NewRequest(http.MethodPost, "/api/like/"+strconv.Itoa(chapter.ID), nil)
	firstReq.RemoteAddr = "203.0.113.1:1234"
	firstReq.Header.Set("User-Agent", "Agent/1")
	firstRec := httptest.NewRecorder()
	api.LikeChapter(firstRec, firstReq)

	if firstRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", firstRec.Code)
	}
	if ct := firstRec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("expected json content type, got %q", ct)
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/api/like/"+strconv.Itoa(chapter.ID), nil)
	secondReq.RemoteAddr = "203.0.113.1:4321"
	secondReq.Header.Set("User-Agent", "Agent/1")
	secondRec := httptest.NewRecorder()
	api.LikeChapter(secondRec, secondReq)

	var payload struct {
		Count int  `json:"count"`
		Liked bool `json:"liked"`
	}
	if err := json.Unmarshal(secondRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if payload.Count != 1 || payload.Liked {
		t.Fatalf("expected duplicate response {count:1, liked:false}, got %+v", payload)
	}
	if mailer.calls != 1 {
		t.Fatalf("expected one milestone mail for first like only, got %d", mailer.calls)
	}
}

func TestLikeAPIUnknownChapterReturns404(t *testing.T) {
	database, cfg, bundle := newPhase4TestEnv(t)
	api := NewAPIHandler(database, cfg, bundle, &testMilestoneMailer{})

	req := httptest.NewRequest(http.MethodPost, "/api/like/99999", nil)
	rec := httptest.NewRecorder()
	api.LikeChapter(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func newPhase4TestEnv(t *testing.T) (*sql.DB, *config.Config, *i18n.Bundle) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "phase4.db")
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
		AdminEmail:    "admin@example.com",
	}
	return database, cfg, bundle
}

func withSessionCookie(t *testing.T, req *http.Request, email, secret string) {
	t.Helper()
	rec := httptest.NewRecorder()
	auth.CreateSession(rec, email, secret)
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie")
	}
	req.AddCookie(cookies[0])
}
