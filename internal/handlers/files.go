package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/Falasefemi2/fileupload/internal/auth"
	"github.com/Falasefemi2/fileupload/internal/models"
)

type initUploadReq struct {
	Name     string  `json:"name"`
	MimeType string  `json:"mimeType"`
	FolderID *string `json:"folderId"`
	Size     *int64  `json:"size"`
}

type initUploadResp struct {
	FileID     string `json:"fileId"`
	StorageKey string `json:"storageKey"`
}

func (h *Handlers) InitUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req initUploadReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.MimeType == "" {
		req.MimeType = "application/octet-stream"
	}

	if req.FolderID != nil && *req.FolderID != "" {
		var exists bool
		err := h.db.QueryRowContext(r.Context(),
			`SELECT EXISTS(SELECT 1 FROM folders WHERE id = $1 AND user_id = $2)`, *req.FolderID, userID).Scan(&exists)
		if err != nil {
			http.Error(w, "failed to validate folder", http.StatusInternalServerError)
			return
		}
		if !exists {
			http.Error(w, "folder not found", http.StatusNotFound)
			return
		}
	} else {
		req.FolderID = nil
	}

	id := uuid.NewString()
	storageKey := fmt.Sprintf("%s/%s", userID, id)
	size := int64(0)
	if req.Size != nil {
		size = *req.Size
	}

	var folderID sql.NullString
	if req.FolderID != nil {
		folderID = sql.NullString{String: *req.FolderID, Valid: true}
	}

	var file models.File
	err := h.db.QueryRowContext(r.Context(),
		`INSERT INTO files (id, user_id, folder_id, name, size, mime_type, storage_key, status) VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending') RETURNING id, user_id, folder_id, name, size, mime_type, storage_key, status, created_at`,
		id, userID, folderID, req.Name, size, req.MimeType, storageKey,
	).Scan(&file.ID, &file.UserID, &file.FolderID, &file.Name, &file.Size, &file.MimeType, &file.StorageKey, &file.Status, &file.CreatedAt)
	if err != nil {
		http.Error(w, "failed to init upload: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(initUploadResp{
		FileID:     file.ID,
		StorageKey: file.StorageKey,
	})
}

func (h *Handlers) CompleteUpload(w http.ResponseWriter, r *http.Request) {
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

	var file models.File
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, user_id, folder_id, name, size, mime_type, storage_key, status, created_at FROM files WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&file.ID, &file.UserID, &file.FolderID, &file.Name, &file.Size, &file.MimeType, &file.StorageKey, &file.Status, &file.CreatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to fetch file", http.StatusInternalServerError)
		return
	}

	if h.storage != nil {
		rc, err := h.storage.Download(r.Context(), file.StorageKey)
		if err != nil {
			http.Error(w, "file not found in storage: "+err.Error(), http.StatusBadRequest)
			return
		}
		rc.Close()
	}

	_, err = h.db.ExecContext(r.Context(), `UPDATE files SET status = 'confirmed' WHERE id = $1`, id)
	if err != nil {
		http.Error(w, "failed to confirm file", http.StatusInternalServerError)
		return
	}
	file.Status = "confirmed"

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(file)
}

func (h *Handlers) DownloadFile(w http.ResponseWriter, r *http.Request) {
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

	var file models.File
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, user_id, folder_id, name, size, mime_type, storage_key, status, created_at FROM files WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&file.ID, &file.UserID, &file.FolderID, &file.Name, &file.Size, &file.MimeType, &file.StorageKey, &file.Status, &file.CreatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to fetch file", http.StatusInternalServerError)
		return
	}
	if file.Status != "confirmed" {
		http.Error(w, "file not ready", http.StatusBadRequest)
		return
	}

	rc, err := h.storage.Download(r.Context(), file.StorageKey)
	if err != nil {
		http.Error(w, "failed to download from storage: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.Name))
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, rc); err != nil {
		// client gone; log but not write header again
		return
	}
}

func (h *Handlers) UploadFile(w http.ResponseWriter, r *http.Request) {
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
	var file models.File
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, user_id, storage_key, status FROM files WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&file.ID, &file.UserID, &file.StorageKey, &file.Status)
	if err == sql.ErrNoRows {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to fetch file", http.StatusInternalServerError)
		return
	}
	if file.Status != "pending" {
		http.Error(w, "file already uploaded", http.StatusBadRequest)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 100<<20)
	defer r.Body.Close()
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if err := h.storage.Upload(r.Context(), file.StorageKey, r.Body, contentType); err != nil {
		http.Error(w, "storage upload failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "uploaded", "storageKey": file.StorageKey})
}

func (h *Handlers) DeleteFile(w http.ResponseWriter, r *http.Request) {
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

	var storageKey string
	err := h.db.QueryRowContext(r.Context(),
		`SELECT storage_key FROM files WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&storageKey)
	if err == sql.ErrNoRows {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to fetch file", http.StatusInternalServerError)
		return
	}

	_, err = h.db.ExecContext(r.Context(), `DELETE FROM files WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		http.Error(w, "failed to delete file", http.StatusInternalServerError)
		return
	}

	_ = h.storage.Delete(r.Context(), storageKey)

	w.WriteHeader(http.StatusNoContent)
}
