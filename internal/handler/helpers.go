package handler

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"codeberg.org/crowdware/forgecrowdbook/internal/auth"
	"codeberg.org/crowdware/forgecrowdbook/internal/config"
	"codeberg.org/crowdware/forgecrowdbook/internal/i18n"
	"codeberg.org/crowdware/forgecrowdbook/internal/model"
	"github.com/yuin/goldmark"
)

func currentUser(r *http.Request, db *sql.DB, cfg *config.Config) (*model.User, error) {
	email, ok := auth.GetSession(r, cfg.SessionSecret)
	if !ok {
		return nil, nil
	}
	return model.GetUserByEmail(db, email)
}

func chapterIDFromPath(path string) (int, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if id, err := strconv.Atoi(parts[i]); err == nil {
			return id, nil
		}
	}
	return 0, fmt.Errorf("chapter id not found in path")
}

func renderPage(w http.ResponseWriter, r *http.Request, db *sql.DB, cfg *config.Config, bundle *i18n.Bundle, page string, data map[string]any) {
	nav, err := baseData(r, cfg, bundle, db)
	if err != nil {
		http.Error(w, "failed to build template data", http.StatusInternalServerError)
		return
	}

	if data == nil {
		data = map[string]any{}
	}
	data["Nav"] = nav
	if _, ok := data["Title"]; !ok {
		data["Title"] = "ForgeCrowdBook"
	}

	templateDir := resolveTemplateDir()
	paths := []string{
		filepath.Join(templateDir, "base.html"),
		filepath.Join(templateDir, page+".html"),
	}

	tpl, err := template.ParseFiles(paths...)
	if err != nil {
		http.Error(w, "failed to parse template", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "failed to render template", http.StatusInternalServerError)
		return
	}
}

func resolveTemplateDir() string {
	candidates := []string{
		"templates",
		filepath.Join("..", "..", "templates"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, "base.html")); err == nil {
			return candidate
		}
	}
	return "templates"
}

func renderMarkdown(md string) (template.HTML, error) {
	var out bytes.Buffer
	if err := goldmark.Convert([]byte(md), &out); err != nil {
		return "", fmt.Errorf("compile markdown: %w", err)
	}
	return template.HTML(out.String()), nil
}

func excerpt(text string, max int) string {
	clean := strings.TrimSpace(text)
	if len(clean) <= max {
		return clean
	}
	return clean[:max]
}
