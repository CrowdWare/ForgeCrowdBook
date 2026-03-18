package handler

import (
	"bytes"
	"database/sql"
	"fmt"
	"html"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"codeberg.org/crowdware/forgecrowdbook/internal/auth"
	"codeberg.org/crowdware/forgecrowdbook/internal/config"
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

func safeHTML(text string) string {
	return html.EscapeString(text)
}
