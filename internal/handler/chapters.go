package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"codeberg.org/crowdware/forgecrowdbook/internal/config"
	"codeberg.org/crowdware/forgecrowdbook/internal/i18n"
	"codeberg.org/crowdware/forgecrowdbook/internal/model"
)

type ChaptersHandler struct {
	DB      *sql.DB
	Config  *config.Config
	I18N    *i18n.Bundle
	Fetcher ContentFetcher
}

func NewChaptersHandler(db *sql.DB, cfg *config.Config, bundle *i18n.Bundle, fetcher ContentFetcher) *ChaptersHandler {
	return &ChaptersHandler{
		DB:      db,
		Config:  cfg,
		I18N:    bundle,
		Fetcher: fetcher,
	}
}

func (h *ChaptersHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, activeBook, ok := h.requireUserAndBook(w, r)
	if !ok {
		return
	}

	chapters, err := model.ListChaptersByAuthor(h.DB, user.ID)
	if err != nil {
		http.Error(w, "failed to load chapters", http.StatusInternalServerError)
		return
	}

	var b strings.Builder
	fmt.Fprintf(&b, "<html><body><h1>My Chapters for %s</h1>", safeHTML(activeBook.Title))
	b.WriteString(`<p><a href="/dashboard/chapters/new">New Chapter</a></p>`)
	for _, chapter := range chapters {
		if chapter.BookID != activeBook.ID {
			continue
		}
		source := chapter.SourceURL
		if len(source) > 60 {
			source = source[:60] + "..."
		}
		fmt.Fprintf(
			&b,
			`<article><h2>%s</h2><p>%s</p><p>Status: %s</p><p>%s</p><p><a href="/dashboard/chapters/%d">Preview</a> | <a href="/dashboard/chapters/%d/edit">Edit</a></p></article>`,
			safeHTML(chapter.Title),
			safeHTML(chapter.PathLabel),
			safeHTML(chapter.Status),
			safeHTML(source),
			chapter.ID,
			chapter.ID,
		)
	}
	b.WriteString("</body></html>")
	fmt.Fprint(w, b.String())
}

func (h *ChaptersHandler) New(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, activeBook, ok := h.requireUserAndBook(w, r)
	if !ok {
		return
	}
	h.renderChapterForm(w, activeBook, nil)
}

func (h *ChaptersHandler) PreviewSource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, _, ok := h.requireUserAndBook(w, r); !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	sourceURL := strings.TrimSpace(r.FormValue("source_url"))
	if sourceURL == "" {
		http.Error(w, "source_url is required", http.StatusBadRequest)
		return
	}

	html, err := h.Fetcher.FetchHTML(h.Fetcher.NormalizeURL(sourceURL))
	if err != nil {
		http.Error(w, "failed to fetch source URL", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

func (h *ChaptersHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, activeBook, ok := h.requireUserAndBook(w, r)
	if !ok {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	pathLabel := strings.TrimSpace(r.FormValue("path_label"))
	sourceURL := strings.TrimSpace(r.FormValue("source_url"))
	if title == "" || sourceURL == "" {
		http.Error(w, "title and source_url are required", http.StatusBadRequest)
		return
	}

	sourceURL = h.Fetcher.NormalizeURL(sourceURL)
	chapter, err := model.CreateChapter(h.DB, activeBook.ID, user.ID, title, model.GenerateSlug(title), pathLabel, sourceURL)
	if err != nil {
		http.Error(w, "failed to create chapter", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/dashboard/chapters/%d", chapter.ID), http.StatusSeeOther)
}

func (h *ChaptersHandler) PreviewPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	_, _, ok := h.requireUserAndBook(w, r)
	if !ok {
		return
	}

	chapter, ok := h.loadOwnedChapter(w, r)
	if !ok {
		return
	}

	content := `<p>Content unavailable.</p>`
	if html, err := h.Fetcher.FetchHTML(chapter.SourceURL); err == nil {
		content = html
	}

	var b strings.Builder
	fmt.Fprintf(&b, "<html><body><h1>%s</h1><p>Path: %s</p><p>Status: %s</p>", safeHTML(chapter.Title), safeHTML(chapter.PathLabel), safeHTML(chapter.Status))
	fmt.Fprintf(&b, `<p><a href="/dashboard/chapters/%d/edit">Edit</a></p>`, chapter.ID)
	if chapter.Status == "draft" {
		fmt.Fprintf(&b, `<form method="POST" action="/dashboard/chapters/%d/submit"><button type="submit">Submit for Review</button></form>`, chapter.ID)
	}
	fmt.Fprintf(&b, `<form method="POST" action="/dashboard/chapters/%d/refresh"><button type="submit">Refresh Content</button></form>`, chapter.ID)
	b.WriteString(`<div class="preview">` + content + `</div></body></html>`)
	fmt.Fprint(w, b.String())
}

func (h *ChaptersHandler) Edit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, activeBook, ok := h.requireUserAndBook(w, r)
	if !ok {
		return
	}

	chapter, ok := h.loadOwnedChapter(w, r)
	if !ok {
		return
	}
	h.renderChapterForm(w, activeBook, chapter)
}

func (h *ChaptersHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, _, ok := h.requireUserAndBook(w, r); !ok {
		return
	}
	chapter, ok := h.loadOwnedChapter(w, r)
	if !ok {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	pathLabel := strings.TrimSpace(r.FormValue("path_label"))
	sourceURL := strings.TrimSpace(r.FormValue("source_url"))
	if title == "" || sourceURL == "" {
		http.Error(w, "title and source_url are required", http.StatusBadRequest)
		return
	}

	sourceURL = h.Fetcher.NormalizeURL(sourceURL)
	if err := model.UpdateChapter(h.DB, chapter.ID, title, pathLabel, sourceURL); err != nil {
		http.Error(w, "failed to update chapter", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/dashboard/chapters/%d", chapter.ID), http.StatusSeeOther)
}

func (h *ChaptersHandler) Submit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, _, ok := h.requireUserAndBook(w, r); !ok {
		return
	}
	chapter, ok := h.loadOwnedChapter(w, r)
	if !ok {
		return
	}

	if err := model.SubmitChapter(h.DB, chapter.ID); err != nil {
		http.Error(w, "failed to submit chapter", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/dashboard/chapters/%d?msg=submitted", chapter.ID), http.StatusSeeOther)
}

func (h *ChaptersHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, _, ok := h.requireUserAndBook(w, r); !ok {
		return
	}
	chapter, ok := h.loadOwnedChapter(w, r)
	if !ok {
		return
	}

	if err := h.Fetcher.Invalidate(chapter.SourceURL); err != nil {
		http.Error(w, "failed to clear cache", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/dashboard/chapters/%d", chapter.ID), http.StatusSeeOther)
}

func (h *ChaptersHandler) renderChapterForm(w http.ResponseWriter, activeBook *model.Book, chapter *model.Chapter) {
	title := ""
	pathLabel := ""
	sourceURL := ""
	action := "/dashboard/chapters"
	button := "Submit for Review"
	if chapter != nil {
		title = chapter.Title
		pathLabel = chapter.PathLabel
		sourceURL = chapter.SourceURL
		action = fmt.Sprintf("/dashboard/chapters/%d", chapter.ID)
		button = "Save"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "<html><body><h1>Chapter Form</h1><p>Active book: %s</p>", safeHTML(activeBook.Title))
	fmt.Fprintf(
		&b,
		`<form method="POST" action="%s">
<label>Title <input type="text" name="title" value="%s" required></label><br>
<label>Path Label <input type="text" name="path_label" value="%s"></label><br>
<label>Source URL <input type="url" name="source_url" value="%s" required></label><br>
<button type="submit">%s</button></form>
<form method="POST" action="/dashboard/preview">
<input type="url" name="source_url" value="%s" required>
<button type="submit">Load Preview</button></form>
</body></html>`,
		safeHTML(action),
		safeHTML(title),
		safeHTML(pathLabel),
		safeHTML(sourceURL),
		safeHTML(button),
		safeHTML(sourceURL),
	)
	fmt.Fprint(w, b.String())
}

func (h *ChaptersHandler) requireUserAndBook(w http.ResponseWriter, r *http.Request) (*model.User, *model.Book, bool) {
	user, err := currentUser(r, h.DB, h.Config)
	if err != nil {
		http.Error(w, "failed to load user", http.StatusInternalServerError)
		return nil, nil, false
	}
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return nil, nil, false
	}

	activeBook, err := GetActiveBook(r, h.DB)
	if err != nil {
		http.Error(w, "failed to load active book", http.StatusInternalServerError)
		return nil, nil, false
	}
	if activeBook == nil {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return nil, nil, false
	}
	return user, activeBook, true
}

func (h *ChaptersHandler) loadOwnedChapter(w http.ResponseWriter, r *http.Request) (*model.Chapter, bool) {
	id, err := chapterIDFromPath(r.URL.Path)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return nil, false
	}

	chapter, err := model.GetChapterByID(h.DB, id)
	if err != nil {
		http.Error(w, "failed to load chapter", http.StatusInternalServerError)
		return nil, false
	}
	if chapter == nil {
		http.NotFound(w, r)
		return nil, false
	}

	user, err := currentUser(r, h.DB, h.Config)
	if err != nil {
		http.Error(w, "failed to load user", http.StatusInternalServerError)
		return nil, false
	}
	if user == nil || chapter.AuthorID != user.ID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false
	}
	return chapter, true
}

func chapterIDPath(path string) int {
	id, _ := strconv.Atoi(pathPart(path, 2))
	return id
}
