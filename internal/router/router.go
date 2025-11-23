package router

import (
	"log/slog"
	"net/http"

	"github.com/thornhall/blog/internal/handler"
	"github.com/thornhall/blog/internal/middleware"
)

func New(h *handler.Handler, log *slog.Logger, publicDir string) http.Handler {
	appMux := http.NewServeMux()
	appMux.HandleFunc("POST /api/likes/{slug}", h.HandleLike)
	appMux.HandleFunc("GET /api/stats/{slug}", h.HandleGetStats)
	appMux.HandleFunc("POST /api/views/{slug}", h.HandleView)

	fs := http.FileServer(http.Dir(publicDir))
	assetsFs := http.FileServer(http.Dir("./assets"))
	appMux.Handle("GET /assets/", http.StripPrefix("/assets/", assetsFs))
	appMux.Handle("GET /", fs)

	// Wrap all routes except SSE in middleware
	var appHandler http.Handler = appMux
	appHandler = middleware.WithLogger(appHandler, log)
	appHandler = middleware.WithRecover(appHandler, log)

	// SSE gets its own handler to avoid middleware which breaks it
	rootMux := http.NewServeMux()
	rootMux.HandleFunc("GET /api/streams/stats", h.HandleStreamStats)
	rootMux.Handle("/", appHandler)

	return rootMux
}
