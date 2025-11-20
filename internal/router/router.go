package router

import (
	"net/http"

	"github.com/thornhall/blog/internal/handler"
)

func New(h *handler.Handler, publicDir string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/likes/{slug}", h.HandleLike)
	mux.HandleFunc("GET /api/stats/{slug}", h.HandleGetStats)
	fs := http.FileServer(http.Dir(publicDir))
	assetsFs := http.FileServer(http.Dir("./assets"))
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", assetsFs))
	mux.Handle("GET /", fs)
	return mux
}
