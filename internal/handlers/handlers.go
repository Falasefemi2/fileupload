package handlers

import (
	"database/sql"

	"golang.org/x/oauth2"

	"github.com/Falasefemi2/fileupload/internal/config"
	"github.com/Falasefemi2/fileupload/internal/storage"
)

type Handlers struct {
	cfg       *config.Config
	db        *sql.DB
	googleCfg *oauth2.Config
	storage   storage.Storage
}

func New(cfg *config.Config, db *sql.DB, googleCfg *oauth2.Config, storage storage.Storage) *Handlers {
	return &Handlers{
		cfg:       cfg,
		db:        db,
		googleCfg: googleCfg,
		storage:   storage,
	}
}
