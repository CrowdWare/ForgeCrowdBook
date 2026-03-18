package handler

import (
	"database/sql"
	"net/http"

	"codeberg.org/crowdware/forgecrowdbook/internal/auth"
	"codeberg.org/crowdware/forgecrowdbook/internal/config"
	"codeberg.org/crowdware/forgecrowdbook/internal/i18n"
)

type HomeHandler struct {
	DB     *sql.DB
	Config *config.Config
	I18N   *i18n.Bundle
}

func NewHomeHandler(db *sql.DB, cfg *config.Config, bundle *i18n.Bundle) *HomeHandler {
	return &HomeHandler{
		DB:     db,
		Config: cfg,
		I18N:   bundle,
	}
}

func (h *HomeHandler) Home(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	renderPage(w, r, h.DB, h.Config, h.I18N, "home", map[string]any{
		"Title": "Home",
	})
}

func (h *HomeHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	auth.ClearSession(w)
	ClearActiveBook(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
