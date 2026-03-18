package main

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
	"codeberg.org/crowdware/forgecrowdbook/internal/fetcher"
	"codeberg.org/crowdware/forgecrowdbook/internal/handler"
	"codeberg.org/crowdware/forgecrowdbook/internal/i18n"
	"codeberg.org/crowdware/forgecrowdbook/internal/model"
	_ "modernc.org/sqlite"
)

type integrationMailer struct {
	sent []string
}

func (m *integrationMailer) SendMagicLink(to, link, lang string, bundle *i18n.Bundle) error {
	m.sent = append(m.sent, "magic:"+to)
	return nil
}
func (m *integrationMailer) SendChapterPublished(to, title, lang string, bundle *i18n.Bundle) error {
	m.sent = append(m.sent, "published:"+to)
	return nil
}
func (m *integrationMailer) SendChapterRejected(to, title, lang string, bundle *i18n.Bundle) error {
	m.sent = append(m.sent, "rejected:"+to)
	return nil
}
func (m *integrationMailer) SendLikeMilestone(to string, count int, title, lang string, bundle *i18n.Bundle) error {
	m.sent = append(m.sent, "milestone:"+to)
	return nil
}

func TestIntegration(t *testing.T) {
	database := openIntegrationDB(t)
	bundle, err := i18n.Load("i18n")
	if err != nil {
		t.Fatalf("i18n.Load failed: %v", err)
	}
	cfg := &config.Config{
		Name:          "ForgeCrowdBook",
		BaseURL:       "http://example.test",
		Port:          "8090",
		SessionSecret: "integration-secret",
		AdminEmail:    "admin@example.com",
	}

	markdownSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("---\n\n> quote\n\n`<code>`"))
	}))
	defer markdownSrv.Close()

	contentFetcher, err := fetcher.New(
		filepath.Join(t.TempDir(), "cache"),
		filepath.Join(t.TempDir(), "cache", "manifest.json"),
		300,
		"https://ipfs.io/ipfs/",
	)
	if err != nil {
		t.Fatalf("fetcher.New failed: %v", err)
	}
	mail := &integrationMailer{}

	mux := buildIntegrationMux(database, cfg, bundle, contentFetcher, mail)

	// TC-01: Register
	registerRec := httptest.NewRecorder()
	registerReq := formRequest(http.MethodPost, "/register", map[string]string{
		"display_name": "Alice",
		"email":        "alice@example.com",
	})
	mux.ServeHTTP(registerRec, registerReq)
	if registerRec.Code != http.StatusOK {
		t.Fatalf("register status: %d body=%q", registerRec.Code, registerRec.Body.String())
	}

	// TC-02: Magic link validation
	var token string
	if err := database.QueryRow(`SELECT token FROM magic_tokens WHERE email = ? ORDER BY id DESC LIMIT 1;`, "alice@example.com").Scan(&token); err != nil {
		t.Fatalf("read token: %v", err)
	}
	authRec := httptest.NewRecorder()
	authReq := httptest.NewRequest(http.MethodGet, "/auth?token="+token, nil)
	mux.ServeHTTP(authRec, authReq)
	if authRec.Code != http.StatusSeeOther || authRec.Header().Get("Location") != "/dashboard" {
		t.Fatalf("auth redirect mismatch: code=%d location=%q", authRec.Code, authRec.Header().Get("Location"))
	}
	userSession := findCookie(authRec.Result().Cookies(), "session")
	if userSession == nil {
		t.Fatal("missing session cookie after auth")
	}

	// TC-03: Select book
	selectRec := httptest.NewRecorder()
	selectReq := httptest.NewRequest(http.MethodPost, "/dashboard/book/1/select", nil)
	selectReq.AddCookie(userSession)
	mux.ServeHTTP(selectRec, selectReq)
	if selectRec.Code != http.StatusSeeOther || selectRec.Header().Get("Location") != "/dashboard/chapters" {
		t.Fatalf("book select mismatch: code=%d location=%q", selectRec.Code, selectRec.Header().Get("Location"))
	}
	activeBook := findCookie(selectRec.Result().Cookies(), "active_book_id")
	if activeBook == nil {
		t.Fatal("missing active_book_id cookie")
	}

	// TC-04: Create chapter with special chars from source URL
	createRec := httptest.NewRecorder()
	createReq := formRequest(http.MethodPost, "/dashboard/chapters", map[string]string{
		"title":      "Special Chapter",
		"path_label": "Path A",
		"source_url": markdownSrv.URL,
	})
	createReq.AddCookie(userSession)
	createReq.AddCookie(activeBook)
	mux.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusSeeOther {
		t.Fatalf("create chapter mismatch: code=%d body=%q", createRec.Code, createRec.Body.String())
	}
	location := createRec.Header().Get("Location")
	if !strings.HasPrefix(location, "/dashboard/chapters/") {
		t.Fatalf("unexpected create redirect location: %q", location)
	}
	chapterID, _ := strconv.Atoi(strings.TrimPrefix(location, "/dashboard/chapters/"))
	if chapterID <= 0 {
		t.Fatalf("invalid chapter id from location %q", location)
	}

	previewRec := httptest.NewRecorder()
	previewReq := httptest.NewRequest(http.MethodGet, location, nil)
	previewReq.AddCookie(userSession)
	previewReq.AddCookie(activeBook)
	mux.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("preview status mismatch: %d body=%q", previewRec.Code, previewRec.Body.String())
	}
	if !strings.Contains(previewRec.Body.String(), "<hr") || !strings.Contains(previewRec.Body.String(), "<blockquote>") {
		t.Fatalf("preview missing rendered markdown: %q", previewRec.Body.String())
	}

	// TC-05: DB stores source_url only
	chapter, err := model.GetChapterByID(database, chapterID)
	if err != nil || chapter == nil {
		t.Fatalf("GetChapterByID failed: %v", err)
	}
	if chapter.SourceURL != markdownSrv.URL {
		t.Fatalf("source_url mismatch: got %q want %q", chapter.SourceURL, markdownSrv.URL)
	}

	// TC-06: Edit chapter page
	editRec := httptest.NewRecorder()
	editReq := httptest.NewRequest(http.MethodGet, "/dashboard/chapters/"+strconv.Itoa(chapterID)+"/edit", nil)
	editReq.AddCookie(userSession)
	editReq.AddCookie(activeBook)
	mux.ServeHTTP(editRec, editReq)
	if editRec.Code != http.StatusOK || !strings.Contains(editRec.Body.String(), markdownSrv.URL) {
		t.Fatalf("edit page mismatch: code=%d body=%q", editRec.Code, editRec.Body.String())
	}

	// TC-07: Submit chapter
	submitRec := httptest.NewRecorder()
	submitReq := httptest.NewRequest(http.MethodPost, "/dashboard/chapters/"+strconv.Itoa(chapterID)+"/submit", nil)
	submitReq.AddCookie(userSession)
	submitReq.AddCookie(activeBook)
	mux.ServeHTTP(submitRec, submitReq)
	if submitRec.Code != http.StatusSeeOther {
		t.Fatalf("submit chapter status mismatch: %d", submitRec.Code)
	}
	chapter, _ = model.GetChapterByID(database, chapterID)
	if chapter.Status != "pending_review" {
		t.Fatalf("expected pending_review, got %q", chapter.Status)
	}

	// TC-08: Admin publish
	adminRec := httptest.NewRecorder()
	auth.CreateSession(adminRec, cfg.AdminEmail, cfg.SessionSecret)
	adminSession := findCookie(adminRec.Result().Cookies(), "session")

	publishRec := httptest.NewRecorder()
	publishReq := httptest.NewRequest(http.MethodPost, "/admin/chapters/"+strconv.Itoa(chapterID)+"/publish", nil)
	publishReq.AddCookie(adminSession)
	mux.ServeHTTP(publishRec, publishReq)
	if publishRec.Code != http.StatusSeeOther {
		t.Fatalf("admin publish status mismatch: %d", publishRec.Code)
	}
	chapter, _ = model.GetChapterByID(database, chapterID)
	if chapter.Status != "published" {
		t.Fatalf("expected published, got %q", chapter.Status)
	}

	// TC-09: Like chapter
	likeReq1 := httptest.NewRequest(http.MethodPost, "/api/like/"+strconv.Itoa(chapterID), nil)
	likeReq1.RemoteAddr = "198.51.100.10:1111"
	likeReq1.Header.Set("User-Agent", "IntegrationTest/1")
	likeRec1 := httptest.NewRecorder()
	mux.ServeHTTP(likeRec1, likeReq1)

	likeReq2 := httptest.NewRequest(http.MethodPost, "/api/like/"+strconv.Itoa(chapterID), nil)
	likeReq2.RemoteAddr = "198.51.100.10:2222"
	likeReq2.Header.Set("User-Agent", "IntegrationTest/1")
	likeRec2 := httptest.NewRecorder()
	mux.ServeHTTP(likeRec2, likeReq2)

	var firstResp, secondResp map[string]any
	_ = json.Unmarshal(likeRec1.Body.Bytes(), &firstResp)
	_ = json.Unmarshal(likeRec2.Body.Bytes(), &secondResp)
	if firstResp["liked"] != true || secondResp["liked"] != false {
		t.Fatalf("unexpected like responses: first=%v second=%v", firstResp, secondResp)
	}
	if int(secondResp["count"].(float64)) != 1 {
		t.Fatalf("expected duplicate count 1, got %v", secondResp["count"])
	}

	// TC-10: Public chapter page
	publicRec := httptest.NewRecorder()
	publicReq := httptest.NewRequest(http.MethodGet, "/books/choose-your-incarnation/"+chapter.Slug, nil)
	mux.ServeHTTP(publicRec, publicReq)
	if publicRec.Code != http.StatusOK || !strings.Contains(publicRec.Body.String(), chapter.Title) {
		t.Fatalf("public page mismatch: code=%d body=%q", publicRec.Code, publicRec.Body.String())
	}
}

func buildIntegrationMux(database *sql.DB, cfg *config.Config, bundle *i18n.Bundle, contentFetcher *fetcher.Fetcher, mail *integrationMailer) *http.ServeMux {
	authHandler := handler.NewAuthHandler(database, cfg, bundle, mail)
	homeHandler := handler.NewHomeHandler(database, cfg, bundle)
	booksHandler := handler.NewBooksHandler(database, cfg, bundle, contentFetcher)
	dashboardHandler := handler.NewDashboardHandler(database, cfg, bundle)
	chaptersHandler := handler.NewChaptersHandler(database, cfg, bundle, contentFetcher)
	adminHandler := handler.NewAdminHandler(database, cfg, bundle, mail)
	apiHandler := handler.NewAPIHandler(database, cfg, bundle, mail)

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	mux.HandleFunc("POST /lang", handler.SetLang)
	mux.HandleFunc("GET /{$}", homeHandler.Home)
	mux.HandleFunc("POST /logout", homeHandler.Logout)
	mux.HandleFunc("GET /books", booksHandler.ListBooks)
	mux.HandleFunc("GET /books/{slug}", booksHandler.BookPage)
	mux.HandleFunc("GET /books/{slug}/{chapter}", booksHandler.ChapterPage)
	mux.HandleFunc("GET /login", authHandler.LoginPage)
	mux.HandleFunc("POST /login", authHandler.Login)
	mux.HandleFunc("GET /register", authHandler.RegisterPage)
	mux.HandleFunc("POST /register", authHandler.Register)
	mux.HandleFunc("GET /auth", authHandler.Auth)
	mux.Handle("GET /dashboard", handler.RequireAuth(cfg, http.HandlerFunc(dashboardHandler.Dashboard)))
	mux.Handle("POST /dashboard/book/{id}/select", handler.RequireAuth(cfg, http.HandlerFunc(dashboardHandler.SelectBook)))
	mux.Handle("GET /dashboard/chapters", handler.RequireAuth(cfg, http.HandlerFunc(chaptersHandler.List)))
	mux.Handle("GET /dashboard/chapters/new", handler.RequireAuth(cfg, http.HandlerFunc(chaptersHandler.New)))
	mux.Handle("POST /dashboard/preview", handler.RequireAuth(cfg, http.HandlerFunc(chaptersHandler.PreviewSource)))
	mux.Handle("POST /dashboard/chapters", handler.RequireAuth(cfg, http.HandlerFunc(chaptersHandler.Create)))
	mux.Handle("GET /dashboard/chapters/{id}", handler.RequireAuth(cfg, http.HandlerFunc(chaptersHandler.PreviewPage)))
	mux.Handle("GET /dashboard/chapters/{id}/edit", handler.RequireAuth(cfg, http.HandlerFunc(chaptersHandler.Edit)))
	mux.Handle("POST /dashboard/chapters/{id}", handler.RequireAuth(cfg, http.HandlerFunc(chaptersHandler.Update)))
	mux.Handle("POST /dashboard/chapters/{id}/submit", handler.RequireAuth(cfg, http.HandlerFunc(chaptersHandler.Submit)))
	mux.Handle("POST /dashboard/chapters/{id}/refresh", handler.RequireAuth(cfg, http.HandlerFunc(chaptersHandler.Refresh)))
	mux.Handle("GET /admin/chapters", handler.RequireAdmin(cfg, http.HandlerFunc(adminHandler.Chapters)))
	mux.Handle("POST /admin/chapters/{id}/publish", handler.RequireAdmin(cfg, http.HandlerFunc(adminHandler.Publish)))
	mux.Handle("POST /admin/chapters/{id}/reject", handler.RequireAdmin(cfg, http.HandlerFunc(adminHandler.Reject)))
	mux.Handle("POST /admin/chapters/{id}/delete", handler.RequireAdmin(cfg, http.HandlerFunc(adminHandler.Delete)))
	mux.Handle("GET /admin/users", handler.RequireAdmin(cfg, http.HandlerFunc(adminHandler.Users)))
	mux.Handle("POST /admin/users/{id}/ban", handler.RequireAdmin(cfg, http.HandlerFunc(adminHandler.BanUser)))
	mux.HandleFunc("POST /api/like/{id}", apiHandler.LikeChapter)
	return mux
}

func openIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	schema := []string{
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL,
			bio TEXT DEFAULT '',
			lang TEXT NOT NULL DEFAULT 'en',
			status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active', 'banned')),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE magic_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL,
			token TEXT NOT NULL UNIQUE,
			expires_at DATETIME NOT NULL,
			used INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE books (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			description TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE chapters (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			book_id INTEGER NOT NULL REFERENCES books(id),
			author_id INTEGER NOT NULL REFERENCES users(id),
			title TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			path_label TEXT DEFAULT '',
			source_url TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft', 'pending_review', 'published', 'rejected')),
			like_count INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			published_at DATETIME
		);`,
		`CREATE TABLE likes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chapter_id INTEGER NOT NULL REFERENCES chapters(id),
			fingerprint TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(chapter_id, fingerprint)
		);`,
		`INSERT INTO books (id, slug, title, description) VALUES
			(1, 'choose-your-incarnation', 'Choose Your Incarnation', 'A collaborative branching narrative.');`,
	}
	for _, stmt := range schema {
		if _, err := database.Exec(stmt); err != nil {
			t.Fatalf("apply schema: %v", err)
		}
	}
	return database
}

func formRequest(method, path string, values map[string]string) *http.Request {
	form := url.Values{}
	for k, v := range values {
		form.Set(k, v)
	}
	req := httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}
