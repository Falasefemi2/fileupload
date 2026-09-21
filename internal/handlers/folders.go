package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/Falasefemi2/fileupload/internal/auth"
	"github.com/Falasefemi2/fileupload/internal/models"
)

type createFolderReq struct {
	Name     string  `json:"name"`
	ParentID *string `json:"parentId"`
}

func (h *Handlers) CreateFolder(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req createFolderReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	if req.ParentID != nil && *req.ParentID != "" {
		var exists bool
		err := h.db.QueryRowContext(r.Context(),
			`SELECT EXISTS(SELECT 1 FROM folders WHERE id = $1 AND user_id = $2)`, *req.ParentID, userID).Scan(&exists)
		if err != nil {
			http.Error(w, "failed to validate parent", http.StatusInternalServerError)
			return
		}
		if !exists {
			http.Error(w, "parent folder not found", http.StatusNotFound)
			return
		}
	} else {
		req.ParentID = nil
	}

	id := uuid.NewString()
	var parentID sql.NullString
	if req.ParentID != nil {
		parentID = sql.NullString{String: *req.ParentID, Valid: true}
	}

	var folder models.Folder
	err := h.db.QueryRowContext(r.Context(),
		`INSERT INTO folders (id, user_id, parent_id, name) VALUES ($1, $2, $3, $4) RETURNING id, user_id, parent_id, name, created_at`,
		id, userID, parentID, req.Name,
	).Scan(&folder.ID, &folder.UserID, &folder.ParentID, &folder.Name, &folder.CreatedAt)
	if err != nil {
		http.Error(w, "failed to create folder: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(folder)
}

func (h *Handlers) ListContents(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	var parentID *string
	if id != "" {
		var exists bool
		err := h.db.QueryRowContext(r.Context(),
			`SELECT EXISTS(SELECT 1 FROM folders WHERE id = $1 AND user_id = $2)`, id, userID).Scan(&exists)
		if err != nil {
			http.Error(w, "failed to verify folder", http.StatusInternalServerError)
			return
		}
		if !exists {
			http.Error(w, "folder not found", http.StatusNotFound)
			return
		}
		parentID = &id
	}

	folders := []models.Folder{}
	files := []models.File{}

	var rows *sql.Rows
	var err error
	if parentID == nil {
		rows, err = h.db.QueryContext(r.Context(),
			`SELECT id, user_id, parent_id, name, created_at FROM folders WHERE user_id = $1 AND parent_id IS NULL ORDER BY name`, userID)
	} else {
		rows, err = h.db.QueryContext(r.Context(),
			`SELECT id, user_id, parent_id, name, created_at FROM folders WHERE user_id = $1 AND parent_id = $2 ORDER BY name`, userID, *parentID)
	}
	if err != nil {
		http.Error(w, "failed to list folders", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var f models.Folder
		if err := rows.Scan(&f.ID, &f.UserID, &f.ParentID, &f.Name, &f.CreatedAt); err != nil {
			http.Error(w, "scan folder failed", http.StatusInternalServerError)
			return
		}
		folders = append(folders, f)
	}

	var fileRows *sql.Rows
	if parentID == nil {
		fileRows, err = h.db.QueryContext(r.Context(),
			`SELECT id, user_id, folder_id, name, size, mime_type, storage_key, status, created_at FROM files WHERE user_id = $1 AND folder_id IS NULL ORDER BY name`, userID)
	} else {
		fileRows, err = h.db.QueryContext(r.Context(),
			`SELECT id, user_id, folder_id, name, size, mime_type, storage_key, status, created_at FROM files WHERE user_id = $1 AND folder_id = $2 ORDER BY name`, userID, *parentID)
	}
	if err != nil {
		http.Error(w, "failed to list files", http.StatusInternalServerError)
		return
	}
	defer fileRows.Close()
	for fileRows.Next() {
		var f models.File
		if err := fileRows.Scan(&f.ID, &f.UserID, &f.FolderID, &f.Name, &f.Size, &f.MimeType, &f.StorageKey, &f.Status, &f.CreatedAt); err != nil {
			http.Error(w, "scan file failed", http.StatusInternalServerError)
			return
		}
		files = append(files, f)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"folders": folders,
		"files":   files,
	})
}

func (h *Handlers) DeleteFolder(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	var exists bool
	err := h.db.QueryRowContext(r.Context(),
		`SELECT EXISTS(SELECT 1 FROM folders WHERE id = $1 AND user_id = $2)`, id, userID).Scan(&exists)
	if err != nil {
		http.Error(w, "failed to verify folder", http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "folder not found", http.StatusNotFound)
		return
	}

	var storageKeys []string
	rows, err := h.db.QueryContext(r.Context(), `
		WITH RECURSIVE subfolders(id) AS (
			SELECT id FROM folders WHERE id = $1 AND user_id = $2
			UNION ALL
			SELECT f.id FROM folders f JOIN subfolders s ON f.parent_id = s.id
		)
		SELECT storage_key FROM files WHERE folder_id IN (SELECT id FROM subfolders) AND user_id = $2
	`, id, userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var k string
			if err := rows.Scan(&k); err == nil {
				storageKeys = append(storageKeys, k)
			}
		}
	}

	_, err = h.db.ExecContext(r.Context(), `DELETE FROM folders WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		http.Error(w, "failed to delete folder: "+err.Error(), http.StatusInternalServerError)
		return
	}

	for _, key := range storageKeys {
		_ = h.storage.Delete(r.Context(), key)
	}

	w.WriteHeader(http.StatusNoContent)
}
