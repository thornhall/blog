package router

import (
	"log/slog"
	"net/http"

	"github.com/thornhall/blog/internal/handler"
	"github.com/thornhall/blog/internal/middleware"
)

func New(h *handler.Handler, log *slog.Logger, publicDir string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/likes/{slug}", h.HandleLike)
	mux.HandleFunc("GET /api/stats/{slug}", h.HandleGetStats)
	fs := http.FileServer(http.Dir(publicDir))
	assetsFs := http.FileServer(http.Dir("./assets"))
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", assetsFs))
	mux.Handle("GET /", fs)
	hnd := middleware.WithLogger(mux, log)
	hnd = middleware.WithRecover(hnd, log)
	return hnd
}
