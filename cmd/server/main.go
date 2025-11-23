package main

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thornhall/blog/internal/backup"
	"github.com/thornhall/blog/internal/db"
	"github.com/thornhall/blog/internal/handler"
	"github.com/thornhall/blog/internal/logging"
	"github.com/thornhall/blog/internal/repo"
	"github.com/thornhall/blog/internal/router"
	"github.com/thornhall/blog/internal/tasks"
	"golang.org/x/crypto/acme/autocert"
)

func NewServer(ctx context.Context, publicDir, domain string) *http.Server {
	logger := logging.New(os.Stdout)
	database := db.New()
	rep := repo.New(database)
	hnd := handler.New(rep, logger, publicDir)
	mux := router.New(hnd, logger, publicDir)

	if _, err := database.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		logger.Error("failed to enable WAL mode", "error", err)
	}

	if domain != "" {
		logger.Info("configuring production server (HTTPS)", "domain", domain)

		certManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(domain, "www."+domain),
			Cache:      autocert.DirCache("certs"),
		}

		go func() {
			logger.Info("starting http redirect server", "addr", ":80")
			if err := http.ListenAndServe(":80", certManager.HTTPHandler(nil)); err != nil {
				logger.Error("redirect server failed", "error", err)
			}
		}()

		return &http.Server{
			Addr:    ":443",
			Handler: mux,
			TLSConfig: &tls.Config{
				GetCertificate: certManager.GetCertificate,
				MinVersion:     tls.VersionTLS12,
			},
			ReadTimeout:       10 * time.Second,
			IdleTimeout:       120 * time.Second,
			WriteTimeout:      0,
			ReadHeaderTimeout: 5 * time.Second,
			BaseContext: func(l net.Listener) context.Context {
				return ctx
			},
		}
	}

	logger.Info("configuring development server (HTTP)", "addr", ":8080")
	return &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		IdleTimeout:       120 * time.Second,
		WriteTimeout:      0,
		ReadHeaderTimeout: 5 * time.Second,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}
}

func main() {
	engineCtx, cancelEngine := context.WithCancel(context.Background())
	defer cancelEngine()

	domain := os.Getenv("DOMAIN")
	srv := NewServer(engineCtx, "./public", domain)

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

	backupCtx, cancelBackup := context.WithCancel(context.Background())
	defer cancelBackup()

	backupClient, err := backup.NewSpaceClient()
	if err != nil {
		log.Printf("error getting S3 client: %v", err)
	} else {
		backupWorker := tasks.NewBackupService(backupClient, "blog.db", time.Hour)
		backupWorker.Start(backupCtx)
	}

	shutDownChan := make(chan os.Signal, 1)
	signal.Notify(shutDownChan, syscall.SIGINT, syscall.SIGTERM)
	<-shutDownChan

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cancelBackup()
	cancelEngine()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("unable to shutdown server gracefully: %v", err)
	}
}
