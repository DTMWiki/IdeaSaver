package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/DTMWiki/IdeaSaver/server/internal/model"
	"github.com/DTMWiki/IdeaSaver/server/internal/repository"
	"github.com/DTMWiki/IdeaSaver/server/internal/storage"
	"github.com/google/uuid"
)

var ErrFileBanned = errors.New("file is banned")

// FileService handles file management business logic.
type FileService struct {
	cfg   *config.Config
	repos *repository.Repositories
	oss   *storage.OSSClient
}

// GetFileByID returns a file owned by the given user.
func (s *FileService) GetFileByID(ctx context.Context, id, userID uuid.UUID) (*model.File, error) {
	file, err := s.repos.Files.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if file.UserID != userID {
		return nil, fmt.Errorf("permission denied")
	}
	if file.DeletedAt != nil {
		return nil, fmt.Errorf("file deleted")
	}
	return file, nil
}

func NewFileService(cfg *config.Config, repos *repository.Repositories, oss *storage.OSSClient) *FileService {
	return &FileService{cfg: cfg, repos: repos, oss: oss}
}

// ListFiles lists files in a directory.
func (s *FileService) ListFiles(ctx context.Context, userID uuid.UUID, parentID *uuid.UUID) ([]model.File, error) {
	return s.repos.Files.ListByParent(ctx, userID, parentID)
}

// CreateDirectory creates a new directory.
func (s *FileService) CreateDirectory(ctx context.Context, userID uuid.UUID, parentID *uuid.UUID, name string) (*model.File, error) {
	name, err := ensureUniqueFileName(ctx, s.repos.Files, userID, parentID, name, nil)
	if err != nil {
		return nil, err
	}
	dir := &model.File{
		UserID:           userID,
		ParentID:         parentID,
		Name:             name,
		IsDirectory:      true,
		ModerationStatus: "normal",
	}
	if err := s.repos.Files.Create(ctx, dir); err != nil {
		return nil, err
	}
	s.logAction(ctx, userID, "create_directory", "file", &dir.ID, map[string]any{
		"name":      dir.Name,
		"parent_id": uuidToString(parentID),
	})
	return dir, nil
}

// Rename renames a file or directory.
func (s *FileService) Rename(ctx context.Context, fileID uuid.UUID, userID uuid.UUID, newName string) error {
	file, err := s.repos.Files.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file.UserID != userID {
		return fmt.Errorf("permission denied")
	}
	newName, err = ensureUniqueFileName(ctx, s.repos.Files, userID, file.ParentID, newName, &fileID)
	if err != nil {
		return err
	}
	if err := s.repos.Files.Rename(ctx, fileID, newName); err != nil {
		return err
	}
	s.logAction(ctx, userID, "rename", "file", &fileID, map[string]any{
		"old_name": file.Name,
		"new_name": newName,
	})
	return nil
}

// Move moves a file to a different directory.
func (s *FileService) Move(ctx context.Context, fileID uuid.UUID, userID uuid.UUID, newParentID *uuid.UUID) error {
	file, err := s.repos.Files.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file.UserID != userID {
		return fmt.Errorf("permission denied")
	}
	if err := s.repos.Files.Move(ctx, fileID, newParentID); err != nil {
		return err
	}
	s.logAction(ctx, userID, "move", "file", &fileID, map[string]any{
		"name":           file.Name,
		"from_parent_id": uuidToString(file.ParentID),
		"to_parent_id":   uuidToString(newParentID),
	})
	return nil
}

// Copy copies a file (creates a new OSS object).
func (s *FileService) Copy(ctx context.Context, fileID uuid.UUID, userID uuid.UUID, destParentID *uuid.UUID) (*model.File, error) {
	src, err := s.repos.Files.FindByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if src.UserID != userID {
		return nil, fmt.Errorf("permission denied")
	}

	newKey := generateStorageKey(userID.String(), src.Name)

	// Copy the object in OSS
	if src.StorageKey != "" {
		if err := s.oss.CopyObject(ctx, src.StorageKey, newKey); err != nil {
			return nil, fmt.Errorf("failed to copy file in storage: %w", err)
		}
	}

	publicURL := fmt.Sprintf("%s/s/%s/%s", s.cfg.PublicBaseURL, userID.String(), filepath.Base(newKey))
	name, err := ensureUniqueFileName(ctx, s.repos.Files, userID, destParentID, src.Name, nil)
	if err != nil {
		return nil, err
	}

	newFile := &model.File{
		UserID:           userID,
		ParentID:         destParentID,
		Name:             name,
		StorageKey:       newKey,
		IsDirectory:      src.IsDirectory,
		MimeType:         src.MimeType,
		Size:             src.Size,
		PublicURL:        publicURL,
		ThumbnailKey:     src.ThumbnailKey,
		ModerationStatus: "normal",
	}

	if err := s.repos.Files.Create(ctx, newFile); err != nil {
		return nil, err
	}

	// Update storage used
	_ = s.repos.Users.UpdateStorageUsed(ctx, userID, src.Size)
	s.logAction(ctx, userID, "copy", "file", &newFile.ID, map[string]any{
		"source_file_id": src.ID.String(),
		"source_name":    src.Name,
		"copied_name":    newFile.Name,
		"parent_id":      uuidToString(destParentID),
	})

	return newFile, nil
}

// SoftDelete moves a file to trash (soft delete).
func (s *FileService) SoftDelete(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) error {
	file, err := s.repos.Files.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file.UserID != userID {
		return fmt.Errorf("permission denied")
	}
	if err := s.repos.Files.SoftDelete(ctx, fileID); err != nil {
		return err
	}
	s.logAction(ctx, userID, "delete", "file", &fileID, map[string]any{
		"name": file.Name,
	})
	return nil
}

// Restore restores a file from trash.
func (s *FileService) Restore(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) error {
	file, err := s.repos.Files.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file.UserID != userID {
		return fmt.Errorf("permission denied")
	}
	if err := s.repos.Files.Restore(ctx, fileID); err != nil {
		return err
	}
	s.logAction(ctx, userID, "restore", "file", &fileID, map[string]any{
		"name": file.Name,
	})
	return nil
}

// PermanentDelete permanently deletes a file and its OSS object.
func (s *FileService) PermanentDelete(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) error {
	file, err := s.repos.Files.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file.UserID != userID {
		return fmt.Errorf("permission denied")
	}

	// Delete from OSS
	if file.StorageKey != "" {
		_ = s.oss.DeleteObject(ctx, file.StorageKey)
	}
	if file.ThumbnailKey != "" {
		_ = s.oss.DeleteObject(ctx, file.ThumbnailKey)
	}

	// Update storage used
	_ = s.repos.Users.UpdateStorageUsed(ctx, userID, -file.Size)
	if err := s.repos.Files.PermanentDelete(ctx, fileID); err != nil {
		return err
	}
	s.logAction(ctx, userID, "permanent_delete", "file", &fileID, map[string]any{
		"name":        file.Name,
		"storage_key": file.StorageKey,
	})
	return nil
}

// ListTrash lists files in the user's trash.
func (s *FileService) ListTrash(ctx context.Context, userID uuid.UUID) ([]model.File, error) {
	return s.repos.Files.ListTrash(ctx, userID)
}

// CleanupTrash removes files that have been in trash for too long.
func (s *FileService) CleanupTrash(ctx context.Context) (int64, error) {
	return s.repos.Files.CleanupTrash(ctx, s.cfg.TrashRetentionDays)
}

// GetFileURL returns the direct link URL and markdown reference for a file.
func (s *FileService) GetFileURL(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) (string, string, error) {
	file, err := s.repos.Files.FindByID(ctx, fileID)
	if err != nil {
		return "", "", err
	}
	if file.UserID != userID {
		return "", "", fmt.Errorf("permission denied")
	}

	url := file.PublicURL
	markdown := fmt.Sprintf("[%s](%s)", file.Name, url)

	// Use image markdown for image files
	if strings.HasPrefix(file.MimeType, "image/") {
		markdown = fmt.Sprintf("![%s](%s)", file.Name, url)
	}

	return url, markdown, nil
}

// ProxyFile returns a reader for the file content from OSS.
func (s *FileService) ProxyFile(ctx context.Context, storageKey string) (io.ReadCloser, string, int64, error) {
	return s.oss.GetObject(ctx, storageKey)
}

// ResolvePublicFile returns a file record by direct-link path segments.
func (s *FileService) ResolvePublicFile(ctx context.Context, userIDStr, filename string) (*model.File, error) {
	return s.repos.Files.FindByStorageKey(ctx, userIDStr+"/"+filename)
}

// SubmitAppeal creates an appeal ticket for a banned file.
func (s *FileService) SubmitAppeal(ctx context.Context, fileID, userID uuid.UUID, reason string) (*model.FileAppeal, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, fmt.Errorf("申诉理由不能为空")
	}
	if len([]rune(reason)) > 1000 {
		return nil, fmt.Errorf("申诉理由不能超过 1000 字")
	}

	file, err := s.repos.Files.FindByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if file.UserID != userID {
		return nil, fmt.Errorf("permission denied")
	}
	if file.ModerationStatus != "banned" {
		return nil, fmt.Errorf("该文件当前未被封禁")
	}

	if _, err := s.repos.FileAppeals.FindPendingByFileID(ctx, fileID); err == nil {
		return nil, fmt.Errorf("该文件已有待处理申诉，请勿重复提交")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	appeal := &model.FileAppeal{
		FileID: fileID,
		UserID: userID,
		Status: "pending",
		Reason: reason,
	}
	if err := s.repos.FileAppeals.Create(ctx, appeal); err != nil {
		return nil, err
	}

	_ = s.repos.AuditLogs.Create(ctx, &model.AuditLog{
		UserID:     userID,
		Action:     "file_appeal_submitted",
		Resource:   "file",
		ResourceID: &fileID,
		Details: map[string]any{
			"reason": reason,
		},
	})

	return appeal, nil
}

// generateStorageKey creates a random storage key for a file.
func generateStorageKey(userID, originalName string) string {
	randBytes := make([]byte, 16)
	_, _ = rand.Read(randBytes)
	randomName := hex.EncodeToString(randBytes)
	ext := filepath.Ext(originalName)
	return userID + "/" + randomName + ext
}

func ensureUniqueFileName(ctx context.Context, repo *repository.FileRepository, userID uuid.UUID, parentID *uuid.UUID, desired string, excludeID *uuid.UUID) (string, error) {
	name := strings.TrimSpace(desired)
	if name == "" {
		return "", fmt.Errorf("名称不能为空")
	}

	base := strings.TrimSuffix(name, filepath.Ext(name))
	ext := filepath.Ext(name)
	if base == "" {
		base = name
		ext = ""
	}

	candidate := name
	index := 2
	for {
		exists, err := repo.ExistsByName(ctx, userID, parentID, candidate, excludeID)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s (%d)%s", base, index, ext)
		index++
	}
}

func (s *FileService) logAction(ctx context.Context, userID uuid.UUID, action, resource string, resourceID *uuid.UUID, details map[string]any) {
	if s == nil || s.repos == nil || s.repos.AuditLogs == nil {
		return
	}
	_ = s.repos.AuditLogs.Create(ctx, &model.AuditLog{
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    details,
	})
}

func uuidToString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}
