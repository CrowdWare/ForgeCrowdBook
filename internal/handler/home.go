package handler

import (
	"database/sql"
	"fmt"
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

	fmt.Fprint(w, `<html><body><h1>ForgeCrowdBook</h1><p>Collaborative branching fiction with chapter moderation.</p><p><a href="/books">Read the book</a></p><p><a href="/register">Write your chapter</a></p></body></html>`)
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
