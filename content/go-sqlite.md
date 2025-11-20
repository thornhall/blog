---
title: Implementing a Hybrid Blog Engine with Go and SQLite
slug: go-sqlite
date: Nov 19, 2025
category: Engineering
excerpt: Outlining my approach to creating this site.
---
> When you have a hammer, everything looks like a nail.

It's no secret that I've been obsessed with Go lately. This year, I've implemented a Discord chat bot, a Grand Exchange clone, 
an in-memory key-value store, and a simple authentication service, all in Go. I even created a VSCode extension that autocompletes
brackets when coding in Go just to save myself a few keystrokes when writing it.

I thought, why stop there? Why not implement a Static Site Generator, like Jekyll, but in Go?

I had two requirements for my SSG:
- Likes and Views are dynamic and persisted (when a user views a post, the views go up, same with likes)
- I can write new posts in a simple markdown format

## The Tech Stack: Go + SQLite + HTML + JavaScript

I wanted to keep my website as lightweight as possible, so I opted to use `SQLite`, an embedded database. What does that actually mean?
It means that the database will live on the same machine my code is running on. Nowadays, it's common for the DB to be a separate process
that your app communicates with over the network. For what I built, such a DB would be overkill.

Go powers the API that receives and serves likes and views. Go also serves the HTML files, such as the one you're looking at right now.

Our app is server rendered, which means all of what you're seeing right now was generated before you even visited this site. Well,
besides one part of the app: the likes and the views count, which are loaded when you visit the page. Being server rendered can speed up
load times on low-end devices as this means the client has less computation to do in order to show you the page.

## The Data Model
Our app is very simple. We have Posts, Likes, and Views. This is the only data needed by the app on the client side:

```go
type Stats struct {
	Slug  string `json:"slug"`
	Likes int    `json:"likes_count"`
	Views int    `json:"views_count"`
}
```

Slug is an extremely nasty word, but for some reason that's what we call the bit of the article that goes in the URL. For example, this site is
`https://thorn.sh/go-sqlite/. In this case, the `go-sqlite` is the slug.

By the way, the syntax highlighting you're seeing in the code snippet? It's also server rendered, using my own custom syntax highlighting theme :)

## The Application Structure

Our server is a Go application, so we follow general Go patterns for the structure:

```go
// cmd (the entrypoints to our app)
// --> builder
// ----> main.go
// --> server
// ----> main.go
//
// internal (the majority of the server code)
// --> db
// --> handler
// --> logging
// --> repo
// --> router
//
// public (the directory where generated HTML is stored)
//
// templates (html template code)
//
// content (the content of the articles, in markdown .md format)
```

Let's walk through a basic HTTP server from the standard library's `net/http` package in the context of this blog's server.

```go
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thornhall/blog/internal/db"
	"github.com/thornhall/blog/internal/handler"
	"github.com/thornhall/blog/internal/logging"
	"github.com/thornhall/blog/internal/repo"
	"github.com/thornhall/blog/internal/router"
	"golang.org/x/crypto/acme/autocert"
)

func NewServer(publicDir, domain string) *http.Server {
	logger := logging.New(os.Stdout)
	database := db.New()
	rep := repo.New(database)
	hnd := handler.New(rep, logger, publicDir)
	mux := router.New(hnd, publicDir)
    
    return &http.Server{
        Addr:    ":443",
        Handler: mux,
        TLSConfig: &tls.Config{
            GetCertificate: certManager.GetCertificate,
            MinVersion:     tls.VersionTLS12,
        },
        ReadTimeout:       10 * time.Second,
        WriteTimeout:      5 * time.Second,
        ReadHeaderTimeout: 5 * time.Second,
	}
}
```

This is how we create a server. For our blog's purposes, we need to pass in a public directory sting and a domain.
The domain is its own topic that I won't dive into here. The public directory is just the path to the `public` directory,
which we know from the file structure is `./public`. 

Note that we also create a logger, database, repo, handler, and router. I'll go over those in the next section.

Next is the `main` function of `main.go` for the server. This function is what executes first when someone runs our Go program.

```go
func main() {
	domain := os.Getenv("DOMAIN")
	srv := NewServer("./public", domain)

    // Start the server in a goroutine so that we can listen for shutdown signals on the main thread.
	go func() {
		var err error
		if domain != "" {
			err = srv.ListenAndServeTLS("", "")
		} else {
			err = srv.ListenAndServe()
		}

		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("unable to start http server: %v", err)
		}
	}()

    // Register listening for shutdown signals
	shutDownChan := make(chan os.Signal, 1)
	signal.Notify(shutDownChan, syscall.SIGINT, syscall.SIGTERM)
	<-shutDownChan // block until a signal is received

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("unable to shutdown server gracefully: %v", err)
	}
}
```

In this code, we start the server in its own thread and listen for shutdown signals on the main thread.
Starting the server in its own goroutine is how we accomplish this.

## The Router

The next most important part of the application to understand is the router. The router is what determines which code gets executed
depending on how you (the client) asks to visit the app.

```go

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
```

This code is telling the server where to find the HTML for the posts and other pages like the homepage. It even tells the server where to find
the media (such as pictures) for the site.

Notice that it takes a handler as a parameter. This is important for understanding how the code works. The router determines which code gets executed
when the user requests a specific part of our app. By visiting this page, your browser did a `GET https://thorn.sh/go-sqlite/`. That happens to be 
default behavior when using a `FileServer`, a standard library package for serving HTML files. But how do we get likes and views? That's where the handler comes in. **The Handler is our custom code that we want executed when the user visits a specific route of our app. Viewing the web page is only one of several routes, defined in the `router`**.


## The Handler
```go
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

	file, err := h.fs.Open(slug + ".html")
	if err != nil {
		HttpErrorResponse(w, "post not found", http.StatusNotFound)
	}
	file.Close()

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
```

I removed some validation logic for security reasons, but here we handle what happens when we receive a like, or when the client
requests the stats of the application. The most important line is the following:

```go
    stats, err := h.repo.IncrementViews(r.Context(), slug) // in HandleGetStats
	stats, err := h.repo.IncrementLikes(r.Context(), slug) // in HandleLikes
```

The handler calls the repo, incrementing the likes when the HandleLikes handler function is invoked, and when the user requests stats,
the Views count is incremented. Now's a perfect time to discuss the repo.

## The Repo

The repo is responsible for interaction with the database. It accepts a db that implements the minimal interface we need for our own usage.

```go
package repo

import (
	"context"
	"github.com/thornhall/blog/internal/db"
)

type Repo struct {
	db db.DB
}

func New(db db.DB) *Repo {
	return &Repo{
		db: db,
	}
}

type Stats struct {
	Slug  string `json:"slug"`
	Likes int    `json:"likes_count"`
	Views int    `json:"views_count"`
}

func (r *Repo) IncrementViews(ctx context.Context, slug string) (Stats, error) {
	var s Stats
	s.Slug = slug

	row := r.db.QueryRowContext(ctx, `
		INSERT INTO post_stats (slug, views, likes) VALUES (?, 1, 0)
		ON CONFLICT(slug) DO UPDATE SET views = views + 1 RETURNING views, likes
	`, slug)
	err := row.Scan(&s.Views, &s.Likes)

	return s, err
}

func (r *Repo) IncrementLikes(ctx context.Context, slug string) (Stats, error) {
	var s Stats
	s.Slug = slug

	row := r.db.QueryRowContext(ctx, "UPDATE post_stats SET likes = likes + 1 WHERE slug = ? RETURNING views, likes", slug)
	err := row.Scan(&s.Views, &s.Likes)

	return s, err
}

func (r *Repo) GetStats(ctx context.Context, slug string) (Stats, error) {
	var s Stats
	s.Slug = slug

	row := r.db.QueryRowContext(ctx, "SELECT views, likes FROM post_stats WHERE slug = ?", slug)
	err := row.Scan(&s.Views, &s.Likes)

	return s, err
}
```

I didn't want to share every line of code of the app for the sake of brevity, but that really covers 90% of what powers this app. However, there's
a key component I've completely skipped: The static site generation.

# Static Site Generation
Up to this point, I've only talked about how the app serves the _already-generated_ HTML files. How do we generate those in the first place? I won't show all the code here for brevity, but the flow for generating a post looks like this:

- Grab the Post template from the `templates` directory.
- Using the `.md` file in the `content` directory matching the slug the user requested, populate the template with the article content.

The template is actually HTML, CSS, and a little bit of JavaScript. In it, we use delimeters to specify where to render the post content on the page.
These delimeters tell the templating engine exactly how to "merge" our template with our actual post.

# Conclusion
That's it. That's most of the app. The question of "how is it deployed" I will leave for another time. I had a blast making this app. In the future I plan to write about projects I've worked on. I'm going to go in-depth about the Grand Exchange clone I built next.