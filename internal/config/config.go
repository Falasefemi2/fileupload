package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port        string
	Env         string
	DatabaseURL string
	FrontendURL string
	JWTSecret   string

	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	SupabaseURL       string
	SupabaseSecretKey string
	SupabaseBucket    string
}

func Load() (Config, error) {
	cfg := Config{
		Port:        getEnv("PORT", "8080"),
		Env:         getEnv("ENV", "development"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		FrontendURL: os.Getenv("FRONTEND_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),

		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),

		SupabaseURL:       os.Getenv("SUPABASE_URL"),
		SupabaseSecretKey: os.Getenv("SUPABASE_SECRET_KEY"),
		SupabaseBucket:    getEnv("SUPABASE_BUCKET", "files"),
	}

	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))

	if value == "" {
		return fallback
	}

	return value
}

func validate(cfg Config) error {
	if err := validatePort(cfg.Port); err != nil {
		return err
	}

	required := map[string]string{
		"DATABASE_URL":         cfg.DatabaseURL,
		"JWT_SECRET":           cfg.JWTSecret,
		"GOOGLE_CLIENT_ID":     cfg.GoogleClientID,
		"GOOGLE_CLIENT_SECRET": cfg.GoogleClientSecret,
		"GOOGLE_REDIRECT_URL":  cfg.GoogleRedirectURL,
		"SUPABASE_URL":         cfg.SupabaseURL,
		"SUPABASE_SECRET_KEY":  cfg.SupabaseSecretKey,
		"SUPABASE_BUCKET":      cfg.SupabaseBucket,
	}

	for key, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", key)
		}
	}

	if len(cfg.JWTSecret) < 32 {
		return errors.New("JWT_SECRET must be at least 32 characters")
	}

	return nil
}

func validatePort(port string) error {
	n, err := strconv.Atoi(port)
	if err != nil {
		return errors.New("PORT must be an integer")
	}

	if n < 1 || n > 65535 {
		return errors.New("PORT must be between 1 and 65535")
	}

	return nil
}
