package middleware

import (
	"log/slog"
	"net/http"

	"github.com/felixge/httpsnoop"
	"github.com/thornhall/blog/internal/handler"
)

// Catches and logs any panics that may occur in HTTP handlers.
func WithRecover(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rcv := recover(); rcv != nil {
				log.Error("panic in http layer", "error", rcv)
				handler.HttpErrorResponse(w, "internal server error", http.StatusInternalServerError)
				return
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Logs information about HTTP requests before passing them to the handler.
func WithLogger(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metrics := httpsnoop.CaptureMetrics(next, w, r)
		log.Info("http response data",
			"method", r.Method,
			"path", r.URL.Path,
			"bytes", metrics.Written,
			"status_code", metrics.Code,
			"duration", metrics.Duration,
		)
	})
}
