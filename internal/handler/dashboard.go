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

type DashboardHandler struct {
	DB     *sql.DB
	Config *config.Config
	I18N   *i18n.Bundle
}

func NewDashboardHandler(db *sql.DB, cfg *config.Config, bundle *i18n.Bundle) *DashboardHandler {
	return &DashboardHandler{
		DB:     db,
		Config: cfg,
		I18N:   bundle,
	}
}

func (h *DashboardHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := currentUser(r, h.DB, h.Config)
	if err != nil {
		http.Error(w, "failed to load user", http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	books, err := model.ListBooks(h.DB)
	if err != nil {
		http.Error(w, "failed to load books", http.StatusInternalServerError)
		return
	}

	active, err := GetActiveBook(r, h.DB)
	if err != nil {
		http.Error(w, "failed to load active book", http.StatusInternalServerError)
		return
	}

	var b strings.Builder
	fmt.Fprintf(&b, "<html><body><h1>Welcome, %s</h1>", safeHTML(user.DisplayName))
	for _, book := range books {
		selected := ""
		if active != nil && active.ID == book.ID {
			selected = " (selected)"
		}
		fmt.Fprintf(
			&b,
			`<article><h2>%s%s</h2><p>%s</p><form method="POST" action="/dashboard/book/%d/select"><button type="submit">Select</button></form></article>`,
			safeHTML(book.Title),
			selected,
			safeHTML(book.Description),
			book.ID,
		)
	}
	if active != nil {
		b.WriteString(`<p><a href="/dashboard/chapters">My Chapters</a></p>`)
	}
	b.WriteString("</body></html>")
	fmt.Fprint(w, b.String())
}

func (h *DashboardHandler) SelectBook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if user, err := currentUser(r, h.DB, h.Config); err != nil {
		http.Error(w, "failed to load user", http.StatusInternalServerError)
		return
	} else if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	bookID, err := strconv.Atoi(pathPart(r.URL.Path, 2))
	if err != nil || bookID <= 0 {
		http.NotFound(w, r)
		return
	}

	book, err := model.GetBookByID(h.DB, bookID)
	if err != nil {
		http.Error(w, "failed to load book", http.StatusInternalServerError)
		return
	}
	if book == nil {
		http.NotFound(w, r)
		return
	}

	SetActiveBook(w, bookID)
	http.Redirect(w, r, "/dashboard/chapters", http.StatusSeeOther)
}
