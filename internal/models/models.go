package models

import "time"

type User struct {
	ID        string    `json:"id"`
	GoogleID  string    `json:"-"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatarUrl"`
	CreatedAt time.Time `json:"createdAt"`
}

type Folder struct {
	ID        string    `json:"id"`
	UserID    string    `json:"-"`
	ParentID  *string   `json:"parentId"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type File struct {
	ID         string    `json:"id"`
	UserID     string    `json:"-"`
	FolderID   *string   `json:"folderId"`
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	MimeType   string    `json:"mimeType"`
	StorageKey string    `json:"-"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"createdAt"`
}
