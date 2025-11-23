package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

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

// GetClientIP extracts the IP and immediately normalizes it.
func GetClientIP(r *http.Request) string {
	// 1. Check Cloudflare/Proxy Header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// XFF can be "client, proxy1, proxy2". We want the first one.
		ips := strings.Split(xff, ",")
		return NormalizeIP(strings.TrimSpace(ips[0]))
	}

	// 2. Check Nginx/Standard Proxy Header
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return NormalizeIP(xrip)
	}

	// 3. Fallback to Direct Connection
	return NormalizeIP(r.RemoteAddr)
}

func NormalizeIP(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err == nil {
		address = host
	}

	ip := net.ParseIP(address)
	if ip == nil {
		return ""
	}

	// IPv4: Return as-is
	if ip4 := ip.To4(); ip4 != nil {
		return ip4.String()
	}

	// IPv6: Mask to the /64 subnet
	mask := net.CIDRMask(64, 128)
	maskedIP := ip.Mask(mask)

	return maskedIP.String()
}

var slugRegex = regexp.MustCompile(`^[a-z0-9-]+$`)

func isValidSlug(slug string) bool {
	return len(slug) > 0 && len(slug) < 100 && slugRegex.MatchString(slug)
}

type ErrorResponse struct {
	Message string `json:"error"`
}

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

	stats, err := h.repo.GetStats(r.Context(), slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			json.NewEncoder(w).Encode(repo.Stats{Slug: slug, Views: 0, Likes: 0})
			return
		}
		h.log.Error("error getting stats", "error", err)
		HttpErrorResponse(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *Handler) HandleView(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if !isValidSlug(slug) {
		HttpErrorResponse(w, "invalid slug format", http.StatusBadRequest)
		return
	}

	ip := GetClientIP(r)
	if ip == "" {
		HttpErrorResponse(w, "invalid request ip", http.StatusBadRequest)
		return
	}

	stats, err := h.repo.IncrementViews(r.Context(), ip, slug)
	if err != nil {
		h.log.Error("error incrementing view", "error", err, "slug", slug)
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

	ip := GetClientIP(r)
	if ip == "" {
		HttpErrorResponse(w, "invalid request ip", http.StatusBadRequest)
		return
	}

	stats, err := h.repo.IncrementLikes(r.Context(), ip, slug)
	if err != nil {
		h.log.Error("error liking post", "error", err)
		HttpErrorResponse(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

var StartTime = time.Now()

type SysStats struct {
	Uptime     string `json:"uptime"`
	MemoryMB   uint64 `json:"memory_mb"`
	Goroutines int    `json:"goroutines"`
	DbSizeMB   string `json:"db_size"`
}

func (h *Handler) HandleStreamStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	flusher.Flush()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	sendStats := func() error {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		var dbSizeStr string
		fileInfo, err := os.Stat("./blog.db")
		if err == nil {
			dbSizeStr = fmt.Sprintf("%.2f", float64(fileInfo.Size())/1024/1024)
		} else {
			dbSizeStr = "0.00"
		}

		stats := SysStats{
			Uptime:     time.Since(StartTime).Round(time.Second).String(),
			MemoryMB:   m.Alloc / 1024 / 1024,
			Goroutines: runtime.NumGoroutine(),
			DbSizeMB:   dbSizeStr,
		}

		data, _ := json.Marshal(stats)

		_, err = fmt.Fprintf(w, "data: %s\n\n", data)
		if err != nil {
			return err
		}

		flusher.Flush()
		return nil
	}

	if err := sendStats(); err != nil {
		return
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if err := sendStats(); err != nil {
				return
			}
		}
	}
}
