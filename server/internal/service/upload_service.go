package service

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/DTMWiki/IdeaSaver/server/internal/model"
	"github.com/DTMWiki/IdeaSaver/server/internal/repository"
	"github.com/DTMWiki/IdeaSaver/server/internal/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

const defaultChunkSize = 5 * 1024 * 1024 // 5MB

// UploadService handles chunked upload with pause/resume/concurrency.
type UploadService struct {
	cfg   *config.Config
	repos *repository.Repositories
	oss   *storage.OSSClient
	sse   *SSEService
	gate  *uploadGate
}

func NewUploadService(cfg *config.Config, repos *repository.Repositories, oss *storage.OSSClient, sse *SSEService) *UploadService {
	maxConc := 10
	if cfg != nil && cfg.MaxConcurrentUploads > 0 {
		maxConc = cfg.MaxConcurrentUploads
	}
	return &UploadService{cfg: cfg, repos: repos, oss: oss, sse: sse, gate: newUploadGate(maxConc)}
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
	if req == nil || req.Size <= 0 {
		return nil, fmt.Errorf("文件大小无效")
	}
	if strings.TrimSpace(req.Filename) == "" {
		return nil, fmt.Errorf("文件名无效")
	}

	// Check max file size before quota (clearer error)
	maxBytes := s.cfg.MaxUploadSizeMB * 1024 * 1024
	if req.Size > maxBytes {
		return nil, fmt.Errorf("文件超过最大允许大小 %d MB", s.cfg.MaxUploadSizeMB)
	}

	// Check quota
	user, err := s.repos.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.StorageUsed+req.Size > user.StorageQuota {
		return nil, fmt.Errorf("存储配额不足，剩余 %d 字节", user.StorageQuota-user.StorageUsed)
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

	// Never trust client-supplied MIME; derive from filename extension only.
	safeMIME := contentTypeForUploadTask(req.Filename)

	// For small files, use single upload (no multipart)
	var uploadID string
	if req.Size > int64(chunkSize) {
		uploadID, err = s.oss.CreateMultipartUpload(ctx, storageKey, safeMIME)
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
	// Local process gate (fast path) + DB lease (cluster-wide across instances).
	if err := s.gate.acquire(ctx); err != nil {
		return err
	}
	defer s.gate.release()

	maxSlots := 10
	if s.cfg != nil && s.cfg.MaxConcurrentUploads > 0 {
		maxSlots = s.cfg.MaxConcurrentUploads
	}
	leaseID, err := s.repos.UploadTasks.TryAcquireChunkLease(ctx, maxSlots, userID.String(), &taskID, 3*time.Minute)
	if err != nil {
		// If leases table is missing (pre-migrate), fall back to process gate only.
		if !strings.Contains(strings.ToLower(err.Error()), "upload_chunk_leases") {
			return fmt.Errorf("获取上传配额失败: %w", err)
		}
	} else if leaseID == 0 {
		return fmt.Errorf("当前上传并发已满，请稍后重试")
	} else {
		defer func() { _ = s.repos.UploadTasks.ReleaseChunkLease(context.Background(), leaseID) }()
	}

	task, err := s.repos.UploadTasks.FindByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task.UserID != userID {
		return ErrPermission
	}
	if task.Status == "paused" {
		return fmt.Errorf("上传已暂停")
	}
	if task.Status == "completed" || task.Status == "completing" || task.Status == "failed" {
		return fmt.Errorf("上传已结束")
	}
	if chunkIndex < 0 || chunkIndex >= task.TotalChunks {
		return fmt.Errorf("无效的分片索引")
	}
	expected := expectedChunkSize(task, chunkIndex)
	if size <= 0 || size != expected {
		return fmt.Errorf("分片大小与声明不符（期望 %d 字节）", expected)
	}
	// Cap the stream so a malicious client cannot over-read past declared size.
	var limited io.Reader = io.LimitReader(body, size)
	if s.cfg != nil && s.cfg.UploadRateLimitMBps > 0 {
		limited = newRateLimitedReader(limited, s.cfg.UploadRateLimitMBps)
	}

	if task.UploadID != "" {
		// Multipart upload: upload part
		partNumber := int32(chunkIndex + 1) // S3 parts are 1-indexed
		// Reject overwrite races on a finished part? S3 allows replace; we still re-record etag.
		etag, err := s.oss.UploadPart(ctx, task.StorageKey, task.UploadID, partNumber, limited, size)
		if err != nil {
			return fmt.Errorf("failed to upload part: %w", err)
		}
		return s.repos.UploadTasks.RecordMultipartPart(ctx, taskID, partNumber, etag)
	}

	// Small file: single put (only chunk 0)
	if chunkIndex != 0 {
		return fmt.Errorf("无效的分片索引")
	}
	if err := s.oss.PutObject(ctx, task.StorageKey, limited, contentTypeForUploadTask(task.Filename), size); err != nil {
		return fmt.Errorf("failed to upload: %w", err)
	}
	return s.repos.UploadTasks.UpdateChunkProgress(ctx, taskID, 1, size)
}

func expectedChunkSize(task *model.UploadTask, chunkIndex int) int64 {
	if task.TotalChunks <= 1 {
		return task.TotalSize
	}
	if chunkIndex < task.TotalChunks-1 {
		return int64(task.ChunkSize)
	}
	// Last chunk: remainder (may equal full chunk when evenly divisible).
	prev := int64(task.ChunkSize) * int64(task.TotalChunks-1)
	last := task.TotalSize - prev
	if last <= 0 {
		return int64(task.ChunkSize)
	}
	return last
}

// PauseUpload pauses an upload task.
func (s *UploadService) PauseUpload(ctx context.Context, taskID uuid.UUID, userID uuid.UUID) error {
	task, err := s.repos.UploadTasks.FindByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task.UserID != userID {
		return ErrPermission
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
		return ErrPermission
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
		return nil, ErrPermission
	}
	// Idempotent: retries must not create duplicate files or re-bill quota.
	if task.Status == "completed" {
		file, findErr := s.repos.Files.FindByStorageKey(ctx, task.StorageKey)
		if findErr != nil {
			return nil, fmt.Errorf("上传已完成，但文件记录不存在")
		}
		return file, nil
	}
	if task.Status == "failed" {
		return nil, fmt.Errorf("上传任务已失败，请重新上传")
	}
	if task.Status == "paused" {
		return nil, fmt.Errorf("上传已暂停，请先恢复后再完成")
	}
	if task.Status == "completing" {
		// Another request is finalizing; surface a soft conflict.
		return nil, fmt.Errorf("上传正在完成，请稍后重试")
	}
	if err := s.assertOwnedDirectory(ctx, userID, parentID); err != nil {
		return nil, err
	}

	// Claim exclusive finalize to prevent double file/quota races.
	claimed, err := s.repos.UploadTasks.ClaimCompleting(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if !claimed {
		// Re-check completed path
		task, err = s.repos.UploadTasks.FindByID(ctx, taskID)
		if err != nil {
			return nil, err
		}
		if task.Status == "completed" {
			file, findErr := s.repos.Files.FindByStorageKey(ctx, task.StorageKey)
			if findErr != nil {
				return nil, fmt.Errorf("上传已完成，但文件记录不存在")
			}
			return file, nil
		}
		if task.Status == "paused" {
			return nil, fmt.Errorf("上传已暂停，请先恢复后再完成")
		}
		return nil, fmt.Errorf("上传正在完成，请稍后重试")
	}

	// Re-load after claim for latest part_etags
	task, err = s.repos.UploadTasks.FindByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Complete multipart upload if applicable
	if task.UploadID != "" {
		var parts []types.CompletedPart
		for i := 1; i <= task.TotalChunks; i++ {
			etag := strings.TrimSpace(task.PartETags[strconv.Itoa(i)])
			if etag == "" {
				_ = s.repos.UploadTasks.UpdateStatus(ctx, taskID, "failed")
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
	} else if task.UploadedChunks < 1 || task.UploadedSize != task.TotalSize {
		_ = s.repos.UploadTasks.UpdateStatus(ctx, taskID, "failed")
		return nil, fmt.Errorf("上传内容不完整")
	}

	// Prefer actual object size from OSS when available.
	finalSize := task.TotalSize
	if s.oss != nil && task.StorageKey != "" {
		if headSize, headErr := s.oss.HeadObjectSize(ctx, task.StorageKey); headErr == nil && headSize > 0 {
			finalSize = headSize
		}
	}

	// Re-check quota at finalize (parallel inits can over-subscribe).
	user, err := s.repos.Users.FindByID(ctx, userID)
	if err != nil {
		_ = s.repos.UploadTasks.UpdateStatus(ctx, taskID, "failed")
		return nil, err
	}
	if user.StorageUsed+finalSize > user.StorageQuota {
		_ = s.repos.UploadTasks.UpdateStatus(ctx, taskID, "failed")
		return nil, fmt.Errorf("存储配额不足，上传无法完成")
	}

	// Create file record
	name, err := ensureUniqueFileName(ctx, s.repos.Files, userID, parentID, task.Filename, nil)
	if err != nil {
		_ = s.repos.UploadTasks.UpdateStatus(ctx, taskID, "failed")
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
		Size:             finalSize,
		PublicURL:        publicURL,
		ModerationStatus: "normal",
	}

	if err := s.createFileWithRetry(ctx, file); err != nil {
		_ = s.repos.UploadTasks.UpdateStatus(ctx, taskID, "failed")
		return nil, err
	}

	_ = s.repos.AuditLogs.Create(ctx, &model.AuditLog{
		UserID:     userID,
		Action:     "upload",
		Resource:   "file",
		ResourceID: &file.ID,
		Details: map[string]any{
			"file_name":   file.Name,
			"storage_key": file.StorageKey,
			"size":        file.Size,
		},
	})

	// Update storage used
	_ = s.repos.Users.UpdateStorageUsed(ctx, userID, finalSize)
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

func (s *UploadService) assertOwnedDirectory(ctx context.Context, userID uuid.UUID, parentID *uuid.UUID) error {
	if parentID == nil {
		return nil
	}
	parent, err := s.repos.Files.FindByID(ctx, *parentID)
	if err != nil {
		return fmt.Errorf("目标文件夹不存在")
	}
	if parent.UserID != userID {
		return ErrPermission
	}
	if parent.DeletedAt != nil {
		return fmt.Errorf("目标文件夹已在回收站中")
	}
	if !parent.IsDirectory {
		return fmt.Errorf("目标必须是文件夹")
	}
	return nil
}

func (s *UploadService) createFileWithRetry(ctx context.Context, file *model.File) error {
	if file == nil {
		return fmt.Errorf("file is nil")
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			nextName, err := ensureUniqueFileName(ctx, s.repos.Files, file.UserID, file.ParentID, file.Name, nil)
			if err != nil {
				return err
			}
			file.Name = nextName
		}

		if err := s.repos.Files.Create(ctx, file); err != nil {
			lastErr = err
			if !isUniqueViolation(err) {
				return err
			}
			continue
		}
		return nil
	}

	if lastErr != nil {
		return lastErr
	}
	return sql.ErrNoRows
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	if pqErr, ok := err.(*pq.Error); ok {
		return string(pqErr.Code) == "23505"
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate key")
}

// ListTasks returns active upload tasks for a user.
func (s *UploadService) ListTasks(ctx context.Context, userID uuid.UUID) ([]model.UploadTask, error) {
	return s.repos.UploadTasks.ListByUser(ctx, userID)
}

func getMimeType(ext string) string {
	// Intentionally never map HTML/SVG/JS/CSS to browser-executable types for storage.
	// Those extensions are stored as octet-stream so same-origin /s/ links cannot XSS.
	mimeTypes := map[string]string{
		".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png",
		".gif": "image/gif", ".webp": "image/webp",
		".mp3": "audio/mpeg", ".wav": "audio/wav", ".ogg": "audio/ogg",
		".mp4": "video/mp4", ".webm": "video/webm", ".mkv": "video/x-matroska",
		".pdf": "application/pdf", ".zip": "application/zip",
		".txt": "text/plain", ".json": "application/json",
		".md": "text/markdown",
		// Dangerous / scriptable — never serve as native browser types from app origin.
		".svg":  "application/octet-stream",
		".html": "application/octet-stream",
		".htm":  "application/octet-stream",
		".js":   "application/octet-stream",
		".mjs":  "application/octet-stream",
		".css":  "application/octet-stream",
		".xml":  "application/octet-stream",
	}
	if mt, ok := mimeTypes[ext]; ok {
		return mt
	}
	return "application/octet-stream"
}

func contentTypeForUploadTask(filename string) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	if ext == "" {
		return "application/octet-stream"
	}
	return getMimeType(ext)
}
