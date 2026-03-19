package service

import (
	"context"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/DTMWiki/IdeaSaver/server/internal/model"
	"github.com/DTMWiki/IdeaSaver/server/internal/repository"
	"github.com/DTMWiki/IdeaSaver/server/internal/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
)

const defaultChunkSize = 5 * 1024 * 1024 // 5MB

// UploadService handles chunked upload with pause/resume/concurrency.
type UploadService struct {
	cfg   *config.Config
	repos *repository.Repositories
	oss   *storage.OSSClient
	sse   *SSEService
}

func NewUploadService(cfg *config.Config, repos *repository.Repositories, oss *storage.OSSClient, sse *SSEService) *UploadService {
	return &UploadService{cfg: cfg, repos: repos, oss: oss, sse: sse}
}

// InitUploadRequest contains parameters for initializing an upload.
type InitUploadRequest struct {
	Filename   string     `json:"filename"`
	Size       int64      `json:"size"`
	MimeType   string     `json:"mime_type"`
	ParentID   *uuid.UUID `json:"parent_id"`
	TargetType string     `json:"target_type"` // "file" or "video"
}

// InitUploadResponse contains the upload task info returned to client.
type InitUploadResponse struct {
	TaskID      uuid.UUID `json:"task_id"`
	ChunkSize   int       `json:"chunk_size"`
	TotalChunks int       `json:"total_chunks"`
}

// InitUpload creates a new chunked upload task.
func (s *UploadService) InitUpload(ctx context.Context, userID uuid.UUID, req *InitUploadRequest) (*InitUploadResponse, error) {
	// Check quota
	user, err := s.repos.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.StorageUsed+req.Size > user.StorageQuota {
		return nil, fmt.Errorf("存储配额不足，剩余 %d 字节", user.StorageQuota-user.StorageUsed)
	}

	// Check max file size
	maxBytes := s.cfg.MaxUploadSizeMB * 1024 * 1024
	if req.Size > maxBytes {
		return nil, fmt.Errorf("文件超过最大允许大小 %d MB", s.cfg.MaxUploadSizeMB)
	}

	targetType := req.TargetType
	if targetType == "" {
		targetType = "file"
	}

	// Generate storage key
	storageKey := generateStorageKey(userID.String(), req.Filename)

	// Calculate chunks
	chunkSize := defaultChunkSize
	totalChunks := int(math.Ceil(float64(req.Size) / float64(chunkSize)))
	if totalChunks == 0 {
		totalChunks = 1
	}

	// For small files, use single upload (no multipart)
	var uploadID string
	if req.Size > int64(chunkSize) {
		uploadID, err = s.oss.CreateMultipartUpload(ctx, storageKey, req.MimeType)
		if err != nil {
			return nil, fmt.Errorf("failed to init multipart upload: %w", err)
		}
	}

	task := &model.UploadTask{
		UserID:      userID,
		Filename:    req.Filename,
		TotalSize:   req.Size,
		ChunkSize:   chunkSize,
		TotalChunks: totalChunks,
		StorageKey:  storageKey,
		UploadID:    uploadID,
		Status:      "pending",
		TargetType:  targetType,
	}

	if err := s.repos.UploadTasks.Create(ctx, task); err != nil {
		return nil, err
	}

	return &InitUploadResponse{
		TaskID:      task.ID,
		ChunkSize:   chunkSize,
		TotalChunks: totalChunks,
	}, nil
}

// UploadChunk handles a single chunk upload.
func (s *UploadService) UploadChunk(ctx context.Context, taskID uuid.UUID, userID uuid.UUID, chunkIndex int, body io.Reader, size int64) error {
	task, err := s.repos.UploadTasks.FindByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task.UserID != userID {
		return fmt.Errorf("permission denied")
	}
	if task.Status == "paused" {
		return fmt.Errorf("上传已暂停")
	}
	if task.Status == "completed" {
		return fmt.Errorf("上传已完成")
	}

	if task.UploadID != "" {
		// Multipart upload: upload part
		partNumber := int32(chunkIndex + 1) // S3 parts are 1-indexed
		etag, err := s.oss.UploadPart(ctx, task.StorageKey, task.UploadID, partNumber, body, size)
		if err != nil {
			return fmt.Errorf("failed to upload part: %w", err)
		}

		newChunks := task.UploadedChunks + 1
		newSize := task.UploadedSize + size
		return s.repos.UploadTasks.UpdateMultipartPart(ctx, taskID, partNumber, etag, newChunks, newSize)
	} else {
		// Small file: single put
		if err := s.oss.PutObject(ctx, task.StorageKey, body, "", size); err != nil {
			return fmt.Errorf("failed to upload: %w", err)
		}

		// Update progress
		newChunks := task.UploadedChunks + 1
		newSize := task.UploadedSize + size
		return s.repos.UploadTasks.UpdateChunkProgress(ctx, taskID, newChunks, newSize)
	}
}

// PauseUpload pauses an upload task.
func (s *UploadService) PauseUpload(ctx context.Context, taskID uuid.UUID, userID uuid.UUID) error {
	task, err := s.repos.UploadTasks.FindByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task.UserID != userID {
		return fmt.Errorf("permission denied")
	}
	return s.repos.UploadTasks.UpdateStatus(ctx, taskID, "paused")
}

// ResumeUpload resumes a paused upload task.
func (s *UploadService) ResumeUpload(ctx context.Context, taskID uuid.UUID, userID uuid.UUID) error {
	task, err := s.repos.UploadTasks.FindByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task.UserID != userID {
		return fmt.Errorf("permission denied")
	}
	if task.Status != "paused" {
		return fmt.Errorf("任务不在暂停状态")
	}
	return s.repos.UploadTasks.UpdateStatus(ctx, taskID, "uploading")
}

// CompleteUpload finalizes a chunked upload.
func (s *UploadService) CompleteUpload(ctx context.Context, taskID uuid.UUID, userID uuid.UUID, parentID *uuid.UUID) (*model.File, error) {
	task, err := s.repos.UploadTasks.FindByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task.UserID != userID {
		return nil, fmt.Errorf("permission denied")
	}

	// Complete multipart upload if applicable
	if task.UploadID != "" {
		var parts []types.CompletedPart
		for i := 1; i <= task.TotalChunks; i++ {
			etag := strings.TrimSpace(task.PartETags[strconv.Itoa(i)])
			if etag == "" {
				return nil, fmt.Errorf("missing multipart etag for part %d", i)
			}
			parts = append(parts, types.CompletedPart{
				PartNumber: aws.Int32(int32(i)),
				ETag:       aws.String(etag),
			})
		}
		if err := s.oss.CompleteMultipartUpload(ctx, task.StorageKey, task.UploadID, parts); err != nil {
			_ = s.repos.UploadTasks.UpdateStatus(ctx, taskID, "failed")
			return nil, fmt.Errorf("failed to complete multipart upload: %w", err)
		}
	}

	// Create file record
	name, err := ensureUniqueFileName(ctx, s.repos.Files, userID, parentID, task.Filename, nil)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(task.Filename))
	mimeType := getMimeType(ext)
	publicURL := fmt.Sprintf("%s/s/%s/%s", s.cfg.PublicBaseURL, userID.String(), filepath.Base(task.StorageKey))

	file := &model.File{
		UserID:           userID,
		ParentID:         parentID,
		Name:             name,
		StorageKey:       task.StorageKey,
		MimeType:         mimeType,
		Size:             task.TotalSize,
		PublicURL:        publicURL,
		ModerationStatus: "normal",
	}

	if err := s.repos.Files.Create(ctx, file); err != nil {
		_ = s.repos.UploadTasks.UpdateStatus(ctx, taskID, "failed")
		return nil, err
	}

	// Update storage used
	_ = s.repos.Users.UpdateStorageUsed(ctx, userID, task.TotalSize)
	_ = s.repos.UploadTasks.UpdateStatus(ctx, taskID, "completed")

	// Push SSE event
	s.sse.SendToUser(userID, SSEEvent{
		Type: "upload_complete",
		Data: map[string]any{
			"file_id":  file.ID,
			"filename": file.Name,
			"url":      publicURL,
			"size":     file.Size,
		},
	})

	return file, nil
}

// ListTasks returns active upload tasks for a user.
func (s *UploadService) ListTasks(ctx context.Context, userID uuid.UUID) ([]model.UploadTask, error) {
	return s.repos.UploadTasks.ListByUser(ctx, userID)
}

func getMimeType(ext string) string {
	mimeTypes := map[string]string{
		".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png",
		".gif": "image/gif", ".webp": "image/webp", ".svg": "image/svg+xml",
		".mp3": "audio/mpeg", ".wav": "audio/wav", ".ogg": "audio/ogg",
		".mp4": "video/mp4", ".webm": "video/webm", ".mkv": "video/x-matroska",
		".pdf": "application/pdf", ".zip": "application/zip",
		".txt": "text/plain", ".html": "text/html", ".css": "text/css",
		".js": "text/javascript", ".json": "application/json",
		".md": "text/markdown",
	}
	if mt, ok := mimeTypes[ext]; ok {
		return mt
	}
	return "application/octet-stream"
}
