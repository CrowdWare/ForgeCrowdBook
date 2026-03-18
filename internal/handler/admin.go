package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"codeberg.org/crowdware/forgecrowdbook/internal/auth"
	"codeberg.org/crowdware/forgecrowdbook/internal/config"
	"codeberg.org/crowdware/forgecrowdbook/internal/i18n"
	"codeberg.org/crowdware/forgecrowdbook/internal/model"
)

type AdminMailer interface {
	SendChapterPublished(to, title, lang string, bundle *i18n.Bundle) error
	SendChapterRejected(to, title, lang string, bundle *i18n.Bundle) error
}

type AdminHandler struct {
	DB     *sql.DB
	Config *config.Config
	I18N   *i18n.Bundle
	Mailer AdminMailer
}

func NewAdminHandler(db *sql.DB, cfg *config.Config, bundle *i18n.Bundle, mailer AdminMailer) *AdminHandler {
	return &AdminHandler{
		DB:     db,
		Config: cfg,
		I18N:   bundle,
		Mailer: mailer,
	}
}

func (h *AdminHandler) Chapters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.requireAdminUser(w, r) {
		return
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	dateFrom := strings.TrimSpace(r.URL.Query().Get("date"))

	query := `
		SELECT c.id, c.title, u.display_name, b.title, c.path_label, c.status, c.like_count, c.created_at
		FROM chapters c
		JOIN users u ON u.id = c.author_id
		JOIN books b ON b.id = c.book_id
		WHERE 1=1
	`
	args := []any{}
	if status != "" && status != "all" {
		query += " AND c.status = ?"
		args = append(args, status)
	}
	if dateFrom != "" {
		query += " AND date(c.created_at) >= date(?)"
		args = append(args, dateFrom)
	}
	query += " ORDER BY c.created_at DESC;"

	rows, err := h.DB.Query(query, args...)
	if err != nil {
		http.Error(w, "failed to load chapters", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var b strings.Builder
	b.WriteString("<html><body><h1>Admin Chapters</h1>")
	for rows.Next() {
		var (
			id        int
			title     string
			author    string
			book      string
			pathLabel string
			state     string
			likeCount int
			createdAt time.Time
		)
		if err := rows.Scan(&id, &title, &author, &book, &pathLabel, &state, &likeCount, &createdAt); err != nil {
			http.Error(w, "failed to scan chapter", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(
			&b,
			`<article><h2>%s</h2><p>%s | %s | %s</p><p>Status: %s | Likes: %d | Created: %s</p><p>
<a href="/dashboard/chapters/%d">Preview</a>
<form method="POST" action="/admin/chapters/%d/publish"><button type="submit">Publish</button></form>
<form method="POST" action="/admin/chapters/%d/reject"><button type="submit">Reject</button></form>
<form method="POST" action="/admin/chapters/%d/delete"><button type="submit">Delete</button></form>
</p></article>`,
			safeHTML(title), safeHTML(author), safeHTML(book), safeHTML(pathLabel), safeHTML(state), likeCount, createdAt.Format(time.RFC3339),
			id, id, id, id,
		)
	}
	b.WriteString("</body></html>")
	fmt.Fprint(w, b.String())
}

func (h *AdminHandler) Publish(w http.ResponseWriter, r *http.Request) {
	h.changeChapterStatus(w, r, "published")
}

func (h *AdminHandler) Reject(w http.ResponseWriter, r *http.Request) {
	h.changeChapterStatus(w, r, "rejected")
}

func (h *AdminHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.requireAdminUser(w, r) {
		return
	}

	id, err := strconv.Atoi(pathPart(r.URL.Path, 2))
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}
	if err := model.DeleteChapter(h.DB, id); err != nil {
		http.Error(w, "failed to delete chapter", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/chapters", http.StatusSeeOther)
}

func (h *AdminHandler) Users(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.requireAdminUser(w, r) {
		return
	}

	rows, err := h.DB.Query(`
		SELECT u.id, u.display_name, u.email, u.status, COUNT(c.id) AS chapter_count, u.created_at
		FROM users u
		LEFT JOIN chapters c ON c.author_id = u.id
		GROUP BY u.id
		ORDER BY u.created_at DESC;
	`)
	if err != nil {
		http.Error(w, "failed to load users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var b strings.Builder
	b.WriteString("<html><body><h1>Admin Users</h1>")
	for rows.Next() {
		var (
			id         int
			name       string
			email      string
			status     string
			chapterCnt int
			createdAt  time.Time
		)
		if err := rows.Scan(&id, &name, &email, &status, &chapterCnt, &createdAt); err != nil {
			http.Error(w, "failed to scan users", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(
			&b,
			`<article><h2>%s</h2><p>%s</p><p>Status: %s | Chapters: %d | Joined: %s</p><form method="POST" action="/admin/users/%d/ban"><button type="submit">Ban</button></form></article>`,
			safeHTML(name), safeHTML(email), safeHTML(status), chapterCnt, createdAt.Format(time.RFC3339), id,
		)
	}
	b.WriteString("</body></html>")
	fmt.Fprint(w, b.String())
}

func (h *AdminHandler) BanUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.requireAdminUser(w, r) {
		return
	}

	id, err := strconv.Atoi(pathPart(r.URL.Path, 2))
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}
	if err := model.BanUser(h.DB, id); err != nil {
		http.Error(w, "failed to ban user", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (h *AdminHandler) changeChapterStatus(w http.ResponseWriter, r *http.Request, state string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !h.requireAdminUser(w, r) {
		return
	}

	id, err := strconv.Atoi(pathPart(r.URL.Path, 2))
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}

	chapter, err := model.GetChapterByID(h.DB, id)
	if err != nil {
		http.Error(w, "failed to load chapter", http.StatusInternalServerError)
		return
	}
	if chapter == nil {
		http.NotFound(w, r)
		return
	}

	switch state {
	case "published":
		err = model.PublishChapter(h.DB, chapter.ID)
	case "rejected":
		err = model.RejectChapter(h.DB, chapter.ID)
	default:
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "failed to update chapter status", http.StatusInternalServerError)
		return
	}

	author, err := model.GetUserByID(h.DB, chapter.AuthorID)
	if err == nil && author != nil && h.Mailer != nil {
		if state == "published" {
			_ = h.Mailer.SendChapterPublished(author.Email, chapter.Title, author.Lang, h.I18N)
		} else {
			_ = h.Mailer.SendChapterRejected(author.Email, chapter.Title, author.Lang, h.I18N)
		}
	}

	http.Redirect(w, r, "/admin/chapters", http.StatusSeeOther)
}

func (h *AdminHandler) requireAdminUser(w http.ResponseWriter, r *http.Request) bool {
	email, ok := auth.GetSession(r, h.Config.SessionSecret)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return false
	}
	if !auth.IsAdmin(email, h.Config.AdminEmail) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return false
	}
	return true
}
