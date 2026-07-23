package service

import (
	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/DTMWiki/IdeaSaver/server/internal/repository"
	"github.com/DTMWiki/IdeaSaver/server/internal/storage"
)

// Services holds all service instances.
type Services struct {
	Auth   *AuthService
	File   *FileService
	Video  *VideoService
	Upload *UploadService
	Share  *ShareService
	Admin  *AdminService
	SSE    *SSEService

	AuditLogs *repository.AuditLogRepository
}

// NewServices creates all service instances.
func NewServices(cfg *config.Config, repos *repository.Repositories, oss *storage.OSSClient, vcloud *storage.VCloudClient) *Services {
	sseService := NewSSEService()
	fileService := NewFileService(cfg, repos, oss)

	return &Services{
		Auth:      NewAuthService(cfg, repos.Users, repos.AuditLogs),
		File:      fileService,
		Video:     NewVideoService(cfg, repos, vcloud, sseService),
		Upload:    NewUploadService(cfg, repos, oss, sseService),
		Share:     NewShareService(cfg, repos, oss),
		Admin:     NewAdminService(cfg, repos, oss, vcloud, fileService),
		SSE:       sseService,
		AuditLogs: repos.AuditLogs,
	}
}
