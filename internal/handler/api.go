package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"codeberg.org/crowdware/forgecrowdbook/internal/config"
	"codeberg.org/crowdware/forgecrowdbook/internal/i18n"
	"codeberg.org/crowdware/forgecrowdbook/internal/mailer"
	"codeberg.org/crowdware/forgecrowdbook/internal/model"
)

type APIMailer interface {
	SendLikeMilestone(to string, count int, title, lang string, bundle *i18n.Bundle) error
}

type APIHandler struct {
	DB     *sql.DB
	Config *config.Config
	I18N   *i18n.Bundle
	Mailer APIMailer
}

func NewAPIHandler(db *sql.DB, cfg *config.Config, bundle *i18n.Bundle, m APIMailer) *APIHandler {
	return &APIHandler{
		DB:     db,
		Config: cfg,
		I18N:   bundle,
		Mailer: m,
	}
}

func (h *APIHandler) LikeChapter(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "method not allowed"})
		return
	}

	chapterID, err := strconv.Atoi(pathPart(r.URL.Path, 2))
	if err != nil || chapterID <= 0 {
		http.NotFound(w, r)
		return
	}

	chapter, err := model.GetChapterByID(h.DB, chapterID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "failed to load chapter"})
		return
	}
	if chapter == nil {
		http.NotFound(w, r)
		return
	}

	newCount, alreadyLiked, err := model.AddLike(h.DB, chapterID, model.Fingerprint(r))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "failed to like chapter"})
		return
	}

	if !alreadyLiked && mailer.IsMilestone(newCount) && h.Mailer != nil {
		author, err := model.GetUserByID(h.DB, chapter.AuthorID)
		if err == nil && author != nil {
			_ = h.Mailer.SendLikeMilestone(author.Email, newCount, chapter.Title, author.Lang, h.I18N)
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"count": newCount,
		"liked": !alreadyLiked,
	})
}
