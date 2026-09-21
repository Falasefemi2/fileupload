package handlers

import (
	"database/sql"

	"golang.org/x/oauth2"

	"github.com/Falasefemi2/fileupload/internal/config"
)

type Handlers struct {
	cfg       *config.Config
	db        *sql.DB
	googleCfg *oauth2.Config
}

func New(cfg *config.Config, db *sql.DB, googleCfg *oauth2.Config) *Handlers {
	return &Handlers{
		cfg:       cfg,
		db:        db,
		googleCfg: googleCfg,
	}
}
