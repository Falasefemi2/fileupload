package main

import (
	"context"
	"log"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/Falasefemi2/fileupload/internal/auth"
	"github.com/Falasefemi2/fileupload/internal/config"
	"github.com/Falasefemi2/fileupload/internal/db"
	"github.com/Falasefemi2/fileupload/internal/handlers"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	database, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	googleCfg := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Endpoint:     google.Endpoint,
		Scopes: []string{
			"openid",
			"email",
			"profile",
		},
	}

	h := handlers.New(&cfg, database, googleCfg)

	mux := http.NewServeMux()

	// Auth routes
	mux.HandleFunc("/auth/google/login", h.GoogleLogin)
	mux.HandleFunc("/auth/google/callback", h.GoogleCallback)
	mux.HandleFunc("/auth/logout", h.Logout)

	// Protected routes
	protected := auth.RequireAuth(cfg.JWTSecret)
	mux.Handle("/auth/me", protected(http.HandlerFunc(h.Me)))

	// Simple health check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	addr := ":" + cfg.Port
	log.Printf("starting server on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
