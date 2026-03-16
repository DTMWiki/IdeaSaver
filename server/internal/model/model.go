package model

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user account synced from Authelia OAuth2.
type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	DisplayName  string    `json:"display_name" db:"display_name"`
	Email        string    `json:"email" db:"email"`
	Role         string    `json:"role" db:"role"` // "user" | "admin"
	StorageQuota int64     `json:"storage_quota" db:"storage_quota"`
	StorageUsed  int64     `json:"storage_used" db:"storage_used"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// File represents a file or directory entry.
type File struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	UserID       uuid.UUID  `json:"user_id" db:"user_id"`
	ParentID     *uuid.UUID `json:"parent_id" db:"parent_id"`
	Name         string     `json:"name" db:"name"`
	StorageKey   string     `json:"storage_key,omitempty" db:"storage_key"`
	IsDirectory  bool       `json:"is_directory" db:"is_directory"`
	MimeType     string     `json:"mime_type,omitempty" db:"mime_type"`
	Size         int64      `json:"size" db:"size"`
	PublicURL    string     `json:"public_url,omitempty" db:"public_url"`
	ThumbnailKey string     `json:"thumbnail_key,omitempty" db:"thumbnail_key"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// Share represents a share link for a file.
type Share struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	FileID    uuid.UUID  `json:"file_id" db:"file_id"`
	Code      string     `json:"code" db:"code"`
	Password  string     `json:"-" db:"password"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	ViewCount int        `json:"view_count" db:"view_count"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// UploadTask represents a chunked upload task for resumable uploads.
type UploadTask struct {
	ID             uuid.UUID `json:"id" db:"id"`
	UserID         uuid.UUID `json:"user_id" db:"user_id"`
	Filename       string    `json:"filename" db:"filename"`
	TotalSize      int64     `json:"total_size" db:"total_size"`
	UploadedSize   int64     `json:"uploaded_size" db:"uploaded_size"`
	ChunkSize      int       `json:"chunk_size" db:"chunk_size"`
	TotalChunks    int       `json:"total_chunks" db:"total_chunks"`
	UploadedChunks int       `json:"uploaded_chunks" db:"uploaded_chunks"`
	StorageKey     string    `json:"storage_key" db:"storage_key"`
	UploadID       string    `json:"upload_id" db:"upload_id"` // S3 multipart upload ID
	Status         string    `json:"status" db:"status"`       // pending/uploading/paused/completed/failed
	TargetType     string    `json:"target_type" db:"target_type"` // file/video
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// Video represents a video managed via DogeCloud VCloud.
type Video struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Title     string    `json:"title" db:"title"`
	VID       string    `json:"vid" db:"vid"`     // DogeCloud video ID
	VCode     string    `json:"vcode" db:"vcode"` // DogeCloud video code
	Status    int16     `json:"status" db:"status"` // 0=disabled 1=enabled
	PlayURL   string    `json:"play_url,omitempty" db:"play_url"`
	Size      int64     `json:"size" db:"size"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// AuditLog represents an audit log entry.
type AuditLog struct {
	ID         int64     `json:"id" db:"id"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	Action     string    `json:"action" db:"action"`
	Resource   string    `json:"resource,omitempty" db:"resource"`
	ResourceID *uuid.UUID `json:"resource_id,omitempty" db:"resource_id"`
	Details    any       `json:"details,omitempty" db:"details"`
	IPAddress  string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent  string    `json:"user_agent,omitempty" db:"user_agent"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`

	// Joined fields (not stored in audit_logs table)
	Username string `json:"username,omitempty" db:"username"`
}
