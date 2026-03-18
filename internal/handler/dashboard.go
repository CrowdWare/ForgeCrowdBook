package handler

import (
	"database/sql"
	"net/http"
	"strconv"

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

	renderPage(w, r, h.DB, h.Config, h.I18N, "dashboard", map[string]any{
		"Title":      "Dashboard",
		"User":       user,
		"Books":      books,
		"ActiveBook": active,
	})
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
