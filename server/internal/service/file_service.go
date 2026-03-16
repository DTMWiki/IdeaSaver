package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

// FileService handles file management business logic.
type FileService struct {
	cfg   *config.Config
	repos *repository.Repositories
	oss   *storage.OSSClient
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
	dir := &model.File{
		UserID:      userID,
		ParentID:    parentID,
		Name:        name,
		IsDirectory: true,
	}
	if err := s.repos.Files.Create(ctx, dir); err != nil {
		return nil, err
	}
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
	return s.repos.Files.Rename(ctx, fileID, newName)
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
	return s.repos.Files.Move(ctx, fileID, newParentID)
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

	newFile := &model.File{
		UserID:       userID,
		ParentID:     destParentID,
		Name:         src.Name,
		StorageKey:   newKey,
		IsDirectory:  src.IsDirectory,
		MimeType:     src.MimeType,
		Size:         src.Size,
		PublicURL:    publicURL,
		ThumbnailKey: src.ThumbnailKey,
	}

	if err := s.repos.Files.Create(ctx, newFile); err != nil {
		return nil, err
	}

	// Update storage used
	_ = s.repos.Users.UpdateStorageUsed(ctx, userID, src.Size)

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
	return s.repos.Files.SoftDelete(ctx, fileID)
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
	return s.repos.Files.Restore(ctx, fileID)
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

	return s.repos.Files.PermanentDelete(ctx, fileID)
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
func (s *FileService) ProxyFile(ctx context.Context, userIDStr, filename string) (io.ReadCloser, string, int64, error) {
	storageKey := userIDStr + "/" + filename
	return s.oss.GetObject(ctx, storageKey)
}

// generateStorageKey creates a random storage key for a file.
func generateStorageKey(userID, originalName string) string {
	randBytes := make([]byte, 16)
	_, _ = rand.Read(randBytes)
	randomName := hex.EncodeToString(randBytes)
	ext := filepath.Ext(originalName)
	return userID + "/" + randomName + ext
}
