package service

import (
	"context"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/DTMWiki/IdeaSaver/server/internal/model"
	"github.com/DTMWiki/IdeaSaver/server/internal/repository"
	"github.com/google/uuid"
)

// AdminService handles admin-only business logic.
type AdminService struct {
	cfg   *config.Config
	repos *repository.Repositories
}

func NewAdminService(cfg *config.Config, repos *repository.Repositories) *AdminService {
	return &AdminService{cfg: cfg, repos: repos}
}

// ListAllFiles returns all files across all users.
func (s *AdminService) ListAllFiles(ctx context.Context, offset, limit int) ([]model.File, int, error) {
	return s.repos.Files.ListAll(ctx, offset, limit)
}

// ListAllVideos returns all videos across all users.
func (s *AdminService) ListAllVideos(ctx context.Context, offset, limit int) ([]model.Video, int, error) {
	return s.repos.Videos.ListAll(ctx, offset, limit)
}

// ListAuditLogs returns audit logs with optional action filter.
func (s *AdminService) ListAuditLogs(ctx context.Context, action string, offset, limit int) ([]model.AuditLog, int, error) {
	return s.repos.AuditLogs.ListAll(ctx, action, offset, limit)
}

// ListUsers returns all users.
func (s *AdminService) ListUsers(ctx context.Context, offset, limit int) ([]model.User, int, error) {
	return s.repos.Users.List(ctx, offset, limit)
}

// UpdateUserQuota sets a user's storage quota.
func (s *AdminService) UpdateUserQuota(ctx context.Context, userID uuid.UUID, quota int64) error {
	return s.repos.Users.UpdateQuota(ctx, userID, quota)
}

// DeleteFile permanently deletes any file (admin).
func (s *AdminService) DeleteFile(ctx context.Context, fileID uuid.UUID) error {
	return s.repos.Files.PermanentDelete(ctx, fileID)
}

// DeleteVideo permanently deletes any video (admin).
func (s *AdminService) DeleteVideo(ctx context.Context, videoID uuid.UUID) error {
	return s.repos.Videos.Delete(ctx, videoID)
}

// CleanupTrash manually cleans up expired trash.
func (s *AdminService) CleanupTrash(ctx context.Context) (int64, error) {
	return s.repos.Files.CleanupTrash(ctx, s.cfg.TrashRetentionDays)
}

// GetUserHistory returns a user's operation history.
func (s *AdminService) GetUserHistory(ctx context.Context, userID uuid.UUID, offset, limit int) ([]model.AuditLog, int, error) {
	return s.repos.AuditLogs.ListByUser(ctx, userID, offset, limit)
}
