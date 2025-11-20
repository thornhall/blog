package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/thornhall/blog/internal/repo"
)

type Handler struct {
	repo *repo.Repo
	log  *slog.Logger
	fs   http.FileSystem
}

func New(repo *repo.Repo, log *slog.Logger, publicDir string) *Handler {
	return &Handler{
		repo: repo,
		log:  log,
		fs:   http.Dir(publicDir),
	}
}

var slugRegex = regexp.MustCompile(`^[a-z0-9-]+$`)

func isValidSlug(slug string) bool {
	return len(slug) > 0 && len(slug) < 100 && slugRegex.MatchString(slug)
}

type ErrorResponse struct {
	Message string `json:"error"`
}

// Used for all error responses for consistency.
func HttpErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	res := ErrorResponse{Message: message}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) HandleGetStats(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	if !isValidSlug(slug) {
		HttpErrorResponse(w, "invalid slug format", http.StatusBadRequest)
		return
	}

	stats, err := h.repo.IncrementViews(r.Context(), slug)
	if err != nil {
		h.log.Error("error incrementing views", "error", err, "slug", slug)
		HttpErrorResponse(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *Handler) HandleLike(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if !isValidSlug(slug) {
		HttpErrorResponse(w, "invalid slug format", http.StatusBadRequest)
		return
	}

	stats, err := h.repo.IncrementLikes(r.Context(), slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			HttpErrorResponse(w, "post not found", http.StatusNotFound)
			return
		}
		h.log.Error("error liking post", "error", err)
		HttpErrorResponse(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
