package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/DTMWiki/IdeaSaver/server/internal/model"
	"github.com/DTMWiki/IdeaSaver/server/internal/repository"
	"github.com/DTMWiki/IdeaSaver/server/internal/storage"
	"github.com/google/uuid"
)

// AdminService handles admin-only business logic.
type AdminService struct {
	cfg   *config.Config
	repos *repository.Repositories
	oss   *storage.OSSClient
}

func NewAdminService(cfg *config.Config, repos *repository.Repositories, oss *storage.OSSClient) *AdminService {
	return &AdminService{cfg: cfg, repos: repos, oss: oss}
}

// ListAllFiles returns all files across all users.
func (s *AdminService) ListAllFiles(ctx context.Context, offset, limit int) ([]model.File, int, error) {
	return s.repos.Files.ListAll(ctx, offset, limit)
}

// ListAllVideos returns all videos across all users.
func (s *AdminService) ListAllVideos(ctx context.Context, offset, limit int) ([]model.Video, int, error) {
	return s.repos.Videos.ListAll(ctx, offset, limit)
}

// ListAuditLogs returns audit logs with optional filters.
func (s *AdminService) ListAuditLogs(ctx context.Context, filter repository.AuditLogFilter, offset, limit int) ([]model.AuditLog, int, error) {
	return s.repos.AuditLogs.ListAll(ctx, filter, offset, limit)
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
func (s *AdminService) DeleteFile(ctx context.Context, fileID, adminID uuid.UUID) error {
	file, err := s.repos.Files.FindByID(ctx, fileID)
	if err != nil {
		return err
	}

	if file.StorageKey != "" {
		_ = s.oss.DeleteObject(ctx, file.StorageKey)
	}
	if file.ThumbnailKey != "" {
		_ = s.oss.DeleteObject(ctx, file.ThumbnailKey)
	}
	_ = s.repos.Users.UpdateStorageUsed(ctx, file.UserID, -file.Size)

	if err := s.repos.Files.PermanentDelete(ctx, fileID); err != nil {
		return err
	}

	_ = s.repos.AuditLogs.Create(ctx, &model.AuditLog{
		UserID:     adminID,
		Action:     "admin_file_deleted",
		Resource:   "file",
		ResourceID: &fileID,
		Details: map[string]any{
			"owner_id":  file.UserID.String(),
			"file_name": file.Name,
		},
	})
	return nil
}

// DeleteVideo permanently deletes any video (admin).
func (s *AdminService) DeleteVideo(ctx context.Context, videoID, adminID uuid.UUID) error {
	video, err := s.repos.Videos.FindByID(ctx, videoID)
	if err != nil {
		return err
	}

	if err := s.repos.Videos.Delete(ctx, videoID); err != nil {
		return err
	}

	_ = s.repos.AuditLogs.Create(ctx, &model.AuditLog{
		UserID:     adminID,
		Action:     "admin_video_deleted",
		Resource:   "video",
		ResourceID: &videoID,
		Details: map[string]any{
			"title":    video.Title,
			"vid":      video.VID,
			"owner_id": video.UserID.String(),
		},
	})
	return nil
}

// CleanupTrash manually cleans up expired trash.
func (s *AdminService) CleanupTrash(ctx context.Context) (int64, error) {
	return s.repos.Files.CleanupTrash(ctx, s.cfg.TrashRetentionDays)
}

// GetUserHistory returns a user's operation history.
func (s *AdminService) GetUserHistory(ctx context.Context, userID uuid.UUID, offset, limit int) ([]model.AuditLog, int, error) {
	return s.repos.AuditLogs.ListByUser(ctx, userID, offset, limit)
}

func (s *AdminService) BanFile(ctx context.Context, fileID, adminID uuid.UUID, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return errors.New("封禁理由不能为空")
	}
	if len([]rune(reason)) > 1000 {
		return errors.New("封禁理由不能超过 1000 字")
	}

	file, err := s.repos.Files.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file.DeletedAt != nil {
		return errors.New("文件已删除")
	}
	if file.IsDirectory {
		return errors.New("暂不支持封禁文件夹")
	}

	if err := s.repos.Files.UpdateModeration(ctx, fileID, "banned", reason, &adminID); err != nil {
		return err
	}

	_ = s.repos.AuditLogs.Create(ctx, &model.AuditLog{
		UserID:     adminID,
		Action:     "file_banned",
		Resource:   "file",
		ResourceID: &fileID,
		Details: map[string]any{
			"owner_id": file.UserID.String(),
			"reason":   reason,
		},
	})
	return nil
}

func (s *AdminService) UnbanFile(ctx context.Context, fileID, adminID uuid.UUID, comment string) error {
	file, err := s.repos.Files.FindByID(ctx, fileID)
	if err != nil {
		return err
	}

	if err := s.repos.Files.UpdateModeration(ctx, fileID, "normal", "", &adminID); err != nil {
		return err
	}

	_ = s.repos.AuditLogs.Create(ctx, &model.AuditLog{
		UserID:     adminID,
		Action:     "file_unbanned",
		Resource:   "file",
		ResourceID: &fileID,
		Details: map[string]any{
			"owner_id": file.UserID.String(),
			"comment":  strings.TrimSpace(comment),
		},
	})
	return nil
}

func (s *AdminService) ListAppeals(ctx context.Context, status string, offset, limit int) ([]model.FileAppeal, int, error) {
	return s.repos.FileAppeals.List(ctx, status, offset, limit)
}

func (s *AdminService) ReviewAppeal(ctx context.Context, appealID, adminID uuid.UUID, decision, comment string) error {
	decision = strings.ToLower(strings.TrimSpace(decision))
	comment = strings.TrimSpace(comment)
	if decision != "approve" && decision != "delete" {
		return errors.New("无效的审核结论，仅支持 approve/delete")
	}

	appeal, err := s.repos.FileAppeals.FindByID(ctx, appealID)
	if err != nil {
		return err
	}
	if appeal.Status != "pending" {
		return errors.New("该工单已处理")
	}

	file, err := s.repos.Files.FindByID(ctx, appeal.FileID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if decision == "approve" {
		if err := s.repos.Files.UpdateModeration(ctx, appeal.FileID, "normal", "", &adminID); err != nil {
			return err
		}
		if err := s.repos.FileAppeals.Review(ctx, appealID, "approved", comment, adminID); err != nil {
			return err
		}
		_ = s.repos.AuditLogs.Create(ctx, &model.AuditLog{
			UserID:     adminID,
			Action:     "file_appeal_approved",
			Resource:   "file_appeal",
			ResourceID: &appealID,
			Details: map[string]any{
				"file_id": appeal.FileID.String(),
				"comment": comment,
			},
		})
		return nil
	}

	if file != nil {
		if file.StorageKey != "" {
			_ = s.oss.DeleteObject(ctx, file.StorageKey)
		}
		if file.ThumbnailKey != "" {
			_ = s.oss.DeleteObject(ctx, file.ThumbnailKey)
		}
		_ = s.repos.Users.UpdateStorageUsed(ctx, file.UserID, -file.Size)
		if err := s.repos.Files.PermanentDelete(ctx, file.ID); err != nil {
			return err
		}
	}

	if err := s.repos.FileAppeals.Review(ctx, appealID, "deleted", comment, adminID); err != nil {
		return err
	}

	_ = s.repos.AuditLogs.Create(ctx, &model.AuditLog{
		UserID:     adminID,
		Action:     "file_appeal_deleted",
		Resource:   "file_appeal",
		ResourceID: &appealID,
		Details: map[string]any{
			"file_id": appeal.FileID.String(),
			"comment": comment,
		},
	})

	return nil
}
