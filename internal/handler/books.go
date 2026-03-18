package handler

import (
	"database/sql"
	"html/template"
	"net/http"
	"strings"

	"codeberg.org/crowdware/forgecrowdbook/internal/config"
	"codeberg.org/crowdware/forgecrowdbook/internal/i18n"
	"codeberg.org/crowdware/forgecrowdbook/internal/model"
)

type ContentFetcher interface {
	FetchHTML(url string) (string, error)
	FetchMarkdown(url string) (string, error)
	NormalizeURL(rawURL string) string
	Invalidate(url string) error
}

type BooksHandler struct {
	DB      *sql.DB
	Config  *config.Config
	I18N    *i18n.Bundle
	Fetcher ContentFetcher
}

func NewBooksHandler(db *sql.DB, cfg *config.Config, bundle *i18n.Bundle, fetcher ContentFetcher) *BooksHandler {
	return &BooksHandler{
		DB:      db,
		Config:  cfg,
		I18N:    bundle,
		Fetcher: fetcher,
	}
}

func (h *BooksHandler) ListBooks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	books, err := model.ListBooks(h.DB)
	if err != nil {
		http.Error(w, "failed to load books", http.StatusInternalServerError)
		return
	}

	renderPage(w, r, h.DB, h.Config, h.I18N, "books", map[string]any{
		"Title": "Books",
		"Books": books,
	})
}

func (h *BooksHandler) BookPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bookSlug := pathPart(r.URL.Path, 1)
	if bookSlug == "" {
		http.NotFound(w, r)
		return
	}

	book, err := model.GetBookBySlug(h.DB, bookSlug)
	if err != nil {
		http.Error(w, "failed to load book", http.StatusInternalServerError)
		return
	}
	if book == nil {
		http.NotFound(w, r)
		return
	}

	chapters, err := model.ListPublishedChaptersByBook(h.DB, book.ID)
	if err != nil {
		http.Error(w, "failed to load chapters", http.StatusInternalServerError)
		return
	}

	type ChapterCard struct {
		ID        int
		Title     string
		Author    string
		PathLabel string
		Excerpt   string
		LikeCount int
		Slug      string
	}
	cards := make([]ChapterCard, 0, len(chapters))
	for _, chapter := range chapters {
		preview := "Content unavailable."
		if md, err := h.Fetcher.FetchMarkdown(chapter.SourceURL); err == nil {
			preview = excerpt(md, 150)
		}
		cards = append(cards, ChapterCard{
			ID:        chapter.ID,
			Title:     chapter.Title,
			Author:    chapter.AuthorName,
			PathLabel: chapter.PathLabel,
			Excerpt:   preview,
			LikeCount: chapter.LikeCount,
			Slug:      chapter.Slug,
		})
	}
	renderPage(w, r, h.DB, h.Config, h.I18N, "book", map[string]any{
		"Title":    book.Title,
		"Book":     book,
		"Chapters": cards,
	})
}

func (h *BooksHandler) ChapterPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bookSlug := pathPart(r.URL.Path, 1)
	chapterSlug := pathPart(r.URL.Path, 2)
	if bookSlug == "" || chapterSlug == "" {
		http.NotFound(w, r)
		return
	}

	book, err := model.GetBookBySlug(h.DB, bookSlug)
	if err != nil {
		http.Error(w, "failed to load book", http.StatusInternalServerError)
		return
	}
	if book == nil {
		http.NotFound(w, r)
		return
	}

	chapter, err := model.GetChapterBySlug(h.DB, chapterSlug)
	if err != nil {
		http.Error(w, "failed to load chapter", http.StatusInternalServerError)
		return
	}
	if chapter == nil || chapter.BookID != book.ID || chapter.Status != "published" {
		http.NotFound(w, r)
		return
	}

	md, err := h.Fetcher.FetchMarkdown(chapter.SourceURL)
	if err != nil {
		http.Error(w, "content unavailable", http.StatusBadGateway)
		return
	}
	compiled, err := renderMarkdown(md)
	if err != nil {
		http.Error(w, "failed to render markdown", http.StatusInternalServerError)
		return
	}

	renderPage(w, r, h.DB, h.Config, h.I18N, "chapter", map[string]any{
		"Title":       chapter.Title,
		"Book":        book,
		"Chapter":     chapter,
		"Rendered":    template.HTML(compiled),
		"OGTitle":     chapter.Title,
		"OGDesc":      excerpt(md, 150),
		"ShareURL":    strings.TrimRight(h.Config.BaseURL, "/") + r.URL.Path,
		"ChapterSlug": chapter.Slug,
	})
}

func pathPart(path string, idx int) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if idx < 0 || idx >= len(parts) {
		return ""
	}
	return parts[idx]
}
