package main

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"codeberg.org/crowdware/forgecrowdbook/internal/config"
	"codeberg.org/crowdware/forgecrowdbook/internal/csrf"
	"codeberg.org/crowdware/forgecrowdbook/internal/db"
	"codeberg.org/crowdware/forgecrowdbook/internal/fetcher"
	"codeberg.org/crowdware/forgecrowdbook/internal/handler"
	"codeberg.org/crowdware/forgecrowdbook/internal/i18n"
	"codeberg.org/crowdware/forgecrowdbook/internal/mailer"
)

var version = "dev"

func main() {
	cfg, err := config.LoadConfig("app.sml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	bundle, err := i18n.Load("i18n")
	if err != nil {
		log.Fatalf("load i18n: %v", err)
	}

	contentFetcher, err := fetcher.New(
		filepath.Join("data", "cache"),
		filepath.Join("data", "cache", "manifest.json"),
		300,
		"https://ipfs.io/ipfs/",
	)
	if err != nil {
		log.Fatalf("create fetcher: %v", err)
	}

	mail := mailer.New(cfg.SMTP)
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

	addr := ":" + cfg.Port
	fmt.Printf("%s version %s listening on %s\n", cfg.Name, version, addr)
	log.Fatal(http.ListenAndServe(addr, csrf.Middleware(mux)))
}
