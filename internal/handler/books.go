package handler

import (
	"database/sql"
	"fmt"
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

	var b strings.Builder
	b.WriteString("<html><body><h1>Books</h1>")
	for _, book := range books {
		fmt.Fprintf(&b, `<article><h2><a href="/books/%s">%s</a></h2><p>%s</p></article>`,
			safeHTML(book.Slug), safeHTML(book.Title), safeHTML(book.Description))
	}
	b.WriteString("</body></html>")
	fmt.Fprint(w, b.String())
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

	var b strings.Builder
	fmt.Fprintf(&b, "<html><body><h1>%s</h1>", safeHTML(book.Title))
	for _, chapter := range chapters {
		preview := "Content unavailable."
		if md, err := h.Fetcher.FetchMarkdown(chapter.SourceURL); err == nil {
			preview = excerpt(md, 150)
		}
		fmt.Fprintf(
			&b,
			`<article><h2>%s</h2><p>By %s</p><p>Path: %s</p><p>%s</p><p>Likes: %d</p><p><a href="/books/%s/%s">Read</a></p></article>`,
			safeHTML(chapter.Title),
			safeHTML(chapter.AuthorName),
			safeHTML(chapter.PathLabel),
			safeHTML(preview),
			chapter.LikeCount,
			safeHTML(book.Slug),
			safeHTML(chapter.Slug),
		)
	}
	b.WriteString("</body></html>")
	fmt.Fprint(w, b.String())
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

	ogDesc := safeHTML(excerpt(md, 150))
	shareURL := strings.TrimRight(h.Config.BaseURL, "/") + r.URL.Path
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	const tpl = `<html><head>
<meta property="og:title" content="{{.Title}}">
<meta property="og:description" content="{{.Description}}">
</head><body>
<p><a href="/books/{{.BookSlug}}">Back to book</a></p>
<h1>{{.Title}}</h1>
<div>{{.Content}}</div>
<form method="POST" action="/api/like/{{.ChapterID}}"><button type="submit">Like</button></form>
<p><a href="https://t.me/share/url?url={{.ShareURL}}">Telegram</a> |
<a href="https://wa.me/?text={{.ShareURL}}">WhatsApp</a> |
<a href="https://www.facebook.com/sharer/sharer.php?u={{.ShareURL}}">Facebook</a> |
<a href="https://x.com/intent/post?url={{.ShareURL}}">X</a></p>
<p>Copy link: {{.ShareURL}}</p>
</body></html>`

	data := struct {
		BookSlug    string
		Title       string
		Description string
		Content     template.HTML
		ChapterID   int
		ShareURL    string
	}{
		BookSlug:    book.Slug,
		Title:       chapter.Title,
		Description: ogDesc,
		Content:     compiled,
		ChapterID:   chapter.ID,
		ShareURL:    shareURL,
	}

	t := template.Must(template.New("chapter").Parse(tpl))
	_ = t.Execute(w, data)
}

func pathPart(path string, idx int) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if idx < 0 || idx >= len(parts) {
		return ""
	}
	return parts[idx]
}
