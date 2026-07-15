package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/DTMWiki/IdeaSaver/server/internal/model"
	"github.com/DTMWiki/IdeaSaver/server/internal/repository"
	"github.com/DTMWiki/IdeaSaver/server/internal/storage"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrSharePasswordRequired indicates the share needs a password and none was provided.
	ErrSharePasswordRequired = errors.New("password_required")
	// ErrSharePasswordInvalid indicates the provided password does not match.
	ErrSharePasswordInvalid = errors.New("password_invalid")
	// ErrShareNotFound indicates the share code does not exist.
	ErrShareNotFound = errors.New("share_not_found")
	// ErrShareExpired indicates the share link is past its expiry.
	ErrShareExpired = errors.New("share_expired")
	// ErrShareBanned indicates the underlying file is banned.
	ErrShareBanned = errors.New("share_banned")
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
// ExpiresIn is seconds from now; 0 means never expires.
type CreateShareRequest struct {
	FileID    uuid.UUID `json:"file_id"`
	Password  string    `json:"password,omitempty"`
	ExpiresIn int       `json:"expires_in,omitempty"`
}

// ShareFileView is a public-safe projection of a shared file (no storage/public URLs).
type ShareFileView struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type,omitempty"`
	IsImage  bool   `json:"is_image"`
}

// CreateShare creates a new share link for a file.
func (s *ShareService) CreateShare(ctx context.Context, userID uuid.UUID, req *CreateShareRequest) (*model.Share, error) {
	file, err := s.repos.Files.FindByID(ctx, req.FileID)
	if err != nil {
		return nil, err
	}
	if file.UserID != userID {
		return nil, ErrPermission
	}
	if file.ModerationStatus == "banned" {
		return nil, fmt.Errorf("该文件已被封禁，无法创建分享")
	}

	code := generateShareCode()

	var expiresAt *time.Time
	if req.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(req.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	hashedPassword, err := hashSharePassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("无法保存分享密码")
	}

	share := &model.Share{
		UserID:    userID,
		FileID:    req.FileID,
		Code:      code,
		Password:  hashedPassword,
		ExpiresAt: expiresAt,
	}

	if err := s.repos.Shares.Create(ctx, share); err != nil {
		return nil, err
	}
	if s.repos.AuditLogs != nil {
		_ = s.repos.AuditLogs.Create(ctx, &model.AuditLog{
			UserID:   userID,
			Action:   "share",
			Resource: "share",
			Details: map[string]any{
				"file_id":    req.FileID.String(),
				"share_id":   share.ID.String(),
				"code":       share.Code,
				"expires_at": share.ExpiresAt,
			},
		})
	}

	// Never return password hash to API callers.
	share.Password = ""
	return share, nil
}

// AccessShare validates a share code/password and returns a public-safe file view.
func (s *ShareService) AccessShare(ctx context.Context, code, password string) (*ShareFileView, error) {
	file, err := s.openSharedFile(ctx, code, password, true)
	if err != nil {
		return nil, err
	}
	return toShareFileView(file), nil
}

// OpenSharedFile validates access and returns the underlying file for download proxying.
func (s *ShareService) OpenSharedFile(ctx context.Context, code, password string) (*model.File, error) {
	return s.openSharedFile(ctx, code, password, false)
}

// ProxySharedFile streams shared file content from storage after access checks.
func (s *ShareService) ProxySharedFile(ctx context.Context, code, password string) (io.ReadCloser, string, int64, *ShareFileView, error) {
	file, err := s.openSharedFile(ctx, code, password, false)
	if err != nil {
		return nil, "", 0, nil, err
	}
	if s.oss == nil {
		return nil, "", 0, nil, fmt.Errorf("storage unavailable")
	}
	reader, contentType, contentLength, err := s.oss.GetObject(ctx, file.StorageKey)
	if err != nil {
		return nil, "", 0, nil, err
	}
	if contentType == "" {
		contentType = file.MimeType
	}
	return reader, contentType, contentLength, toShareFileView(file), nil
}

func (s *ShareService) openSharedFile(ctx context.Context, code, password string, incrementView bool) (*model.File, error) {
	share, err := s.repos.Shares.FindByCode(ctx, code)
	if err != nil {
		return nil, ErrShareNotFound
	}

	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		return nil, ErrShareExpired
	}

	if share.Password != "" {
		if strings.TrimSpace(password) == "" {
			return nil, ErrSharePasswordRequired
		}
		if !checkSharePassword(share.Password, password) {
			return nil, ErrSharePasswordInvalid
		}
	}

	file, err := s.repos.Files.FindByID(ctx, share.FileID)
	if err != nil {
		return nil, ErrShareNotFound
	}
	if file.ModerationStatus == "banned" {
		return nil, ErrShareBanned
	}
	if file.DeletedAt != nil {
		return nil, ErrShareNotFound
	}

	if incrementView {
		_ = s.repos.Shares.IncrementViewCount(ctx, share.ID)
	}

	return file, nil
}

// ListShares returns all shares for a user.
func (s *ShareService) ListShares(ctx context.Context, userID uuid.UUID) ([]model.Share, error) {
	return s.repos.Shares.ListByUser(ctx, userID)
}

// DeleteShare removes a share link.
func (s *ShareService) DeleteShare(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	share, err := s.repos.Shares.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if share.UserID != userID {
		return ErrPermission
	}
	if err := s.repos.Shares.Delete(ctx, id); err != nil {
		return err
	}
	if s.repos.AuditLogs != nil {
		_ = s.repos.AuditLogs.Create(ctx, &model.AuditLog{
			UserID:     userID,
			Action:     "share_delete",
			Resource:   "share",
			ResourceID: &share.ID,
			Details: map[string]any{
				"file_id": share.FileID.String(),
				"code":    share.Code,
			},
		})
	}
	return nil
}

func toShareFileView(file *model.File) *ShareFileView {
	if file == nil {
		return nil
	}
	return &ShareFileView{
		Name:     file.Name,
		Size:     file.Size,
		MimeType: file.MimeType,
		IsImage:  strings.HasPrefix(file.MimeType, "image/"),
	}
}

func generateShareCode() string {
	bytes := make([]byte, 8)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func hashSharePassword(password string) (string, error) {
	password = strings.TrimSpace(password)
	if password == "" {
		return "", nil
	}
	if len([]rune(password)) > 128 {
		return "", fmt.Errorf("密码过长")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func checkSharePassword(stored, provided string) bool {
	if stored == "" {
		return true
	}
	// bcrypt hashes
	if strings.HasPrefix(stored, "$2a$") || strings.HasPrefix(stored, "$2b$") || strings.HasPrefix(stored, "$2y$") {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(provided)) == nil
	}
	// legacy plaintext fallback
	return subtle.ConstantTimeCompare([]byte(stored), []byte(provided)) == 1
}
