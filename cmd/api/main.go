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
	"github.com/Falasefemi2/fileupload/internal/storage"
)

type protectedMux struct {
	mux  *http.ServeMux
	wrap func(http.Handler) http.Handler
}

func (p protectedMux) HandleFunc(pattern string, handler http.HandlerFunc) {
	p.mux.Handle(pattern, p.wrap(handler))
}

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

	storageService := storage.NewSupabaseStorage(
		cfg.SupabaseURL,
		cfg.SupabaseSecretKey,
		cfg.SupabaseBucket,
	)

	h := handlers.New(
		&cfg,
		database,
		googleCfg,
		storageService,
	)

	mux := http.NewServeMux()

	mux.HandleFunc("/auth/google/login", h.GoogleLogin)
	mux.HandleFunc("/auth/google/callback", h.GoogleCallback)
	mux.HandleFunc("/auth/logout", h.Logout)

	protected := protectedMux{
		mux:  mux,
		wrap: auth.RequireAuth(cfg.JWTSecret),
	}
	protected.HandleFunc("/auth/me", h.Me)
	protected.HandleFunc("POST /folders", h.CreateFolder)
	protected.HandleFunc("GET /folders/{id}/contents", h.ListContents)
	protected.HandleFunc("GET /folders/contents", h.ListContents) // root
	protected.HandleFunc("DELETE /folders/{id}", h.DeleteFolder)

	protected.HandleFunc("POST /files/init", h.InitUpload)
	protected.HandleFunc("POST /files/{id}/upload", h.UploadFile)
	protected.HandleFunc("POST /files/{id}/complete", h.CompleteUpload)
	protected.HandleFunc("GET /files/{id}/download", h.DownloadFile)
	protected.HandleFunc("DELETE /files/{id}", h.DeleteFile)

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	corsHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			allowed := cfg.FrontendURL
			if cfg.Env != "production" {
				if origin == "http://localhost:3000" || origin == "http://127.0.0.1:3000" {
					allowed = origin
				}
			}
			if origin != "" && (origin == cfg.FrontendURL || allowed == origin) {
				w.Header().Set("Access-Control-Allow-Origin", allowed)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	addr := ":" + cfg.Port
	log.Printf("starting server on %s (frontend %s env=%s)", addr, cfg.FrontendURL, cfg.Env)
	if err := http.ListenAndServe(addr, corsHandler(mux)); err != nil {
		log.Fatal(err)
	}
}
