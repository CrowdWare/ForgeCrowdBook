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

	type ChapterRow struct {
		ID          int
		Title       string
		PathLabel   string
		Status      string
		SourceShort string
	}
	rows := []ChapterRow{}
	for _, chapter := range chapters {
		if chapter.BookID != activeBook.ID {
			continue
		}
		source := chapter.SourceURL
		if len(source) > 60 {
			source = source[:60] + "..."
		}
		rows = append(rows, ChapterRow{
			ID:          chapter.ID,
			Title:       chapter.Title,
			PathLabel:   chapter.PathLabel,
			Status:      chapter.Status,
			SourceShort: source,
		})
	}
	renderPage(w, r, h.DB, h.Config, h.I18N, "chapters", map[string]any{
		"Title":   "My Chapters",
		"Book":    activeBook,
		"Rows":    rows,
		"HasBook": activeBook != nil,
	})
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
	h.renderChapterForm(w, r, activeBook, nil)
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

	content := "<p>Content unavailable.</p>"
	if html, err := h.Fetcher.FetchHTML(chapter.SourceURL); err == nil {
		content = html
	}
	renderPage(w, r, h.DB, h.Config, h.I18N, "chapter-preview", map[string]any{
		"Title":   chapter.Title,
		"Chapter": chapter,
		"Content": template.HTML(content),
	})
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
	h.renderChapterForm(w, r, activeBook, chapter)
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

func (h *ChaptersHandler) renderChapterForm(w http.ResponseWriter, r *http.Request, activeBook *model.Book, chapter *model.Chapter) {
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

	renderPage(w, r, h.DB, h.Config, h.I18N, "chapter-editor", map[string]any{
		"Title":      "Chapter Editor",
		"ActiveBook": activeBook,
		"Chapter": map[string]any{
			"Title":     title,
			"PathLabel": pathLabel,
			"SourceURL": sourceURL,
		},
		"FormAction": action,
		"ButtonText": button,
		"IsEdit":     chapter != nil,
	})
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
