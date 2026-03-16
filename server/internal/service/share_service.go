package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/DTMWiki/IdeaSaver/server/internal/model"
	"github.com/DTMWiki/IdeaSaver/server/internal/repository"
	"github.com/DTMWiki/IdeaSaver/server/internal/storage"
	"github.com/google/uuid"
)

// ShareService handles share link business logic.
type ShareService struct {
	cfg   *config.Config
	repos *repository.Repositories
	oss   *storage.OSSClient
}

func NewShareService(cfg *config.Config, repos *repository.Repositories, oss *storage.OSSClient) *ShareService {
	return &ShareService{cfg: cfg, repos: repos, oss: oss}
}

// CreateShareRequest contains parameters for creating a share link.
type CreateShareRequest struct {
	FileID    uuid.UUID `json:"file_id"`
	Password  string    `json:"password,omitempty"`
	ExpiresIn int       `json:"expires_in,omitempty"` // hours, 0 = never
}

// CreateShare creates a new share link for a file.
func (s *ShareService) CreateShare(ctx context.Context, userID uuid.UUID, req *CreateShareRequest) (*model.Share, error) {
	// Verify file ownership
	file, err := s.repos.Files.FindByID(ctx, req.FileID)
	if err != nil {
		return nil, err
	}
	if file.UserID != userID {
		return nil, fmt.Errorf("permission denied")
	}

	code := generateShareCode()

	var expiresAt *time.Time
	if req.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(req.ExpiresIn) * time.Hour)
		expiresAt = &t
	}

	share := &model.Share{
		UserID:    userID,
		FileID:    req.FileID,
		Code:      code,
		Password:  req.Password,
		ExpiresAt: expiresAt,
	}

	if err := s.repos.Shares.Create(ctx, share); err != nil {
		return nil, err
	}

	return share, nil
}

// AccessShare retrieves a shared file by its code.
func (s *ShareService) AccessShare(ctx context.Context, code, password string) (*model.File, error) {
	share, err := s.repos.Shares.FindByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("分享链接不存在")
	}

	// Check expiry
	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		return nil, fmt.Errorf("分享链接已过期")
	}

	// Check password
	if share.Password != "" && share.Password != password {
		return nil, fmt.Errorf("密码错误")
	}

	// Increment view count
	_ = s.repos.Shares.IncrementViewCount(ctx, share.ID)

	// Get file
	file, err := s.repos.Files.FindByID(ctx, share.FileID)
	if err != nil {
		return nil, fmt.Errorf("文件不存在")
	}

	return file, nil
}

// ListShares returns all shares for a user.
func (s *ShareService) ListShares(ctx context.Context, userID uuid.UUID) ([]model.Share, error) {
	return s.repos.Shares.ListByUser(ctx, userID)
}

// DeleteShare removes a share link.
func (s *ShareService) DeleteShare(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.repos.Shares.Delete(ctx, id)
}

func generateShareCode() string {
	bytes := make([]byte, 8)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
