package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/DTMWiki/IdeaSaver/server/internal/middleware"
	"github.com/DTMWiki/IdeaSaver/server/internal/service"
	"github.com/gin-gonic/gin"
)

const oauthStateCookie = "ideasaver_oauth_state"
const authTokenCookie = "ideasaver_token"

func SetupRoutes(r *gin.Engine, cfg *config.Config, svc *service.Services) {
	// Public routes (stricter rate limits on auth + provider callbacks)
	authPublic := r.Group("/api/auth")
	authPublic.Use(middleware.StrictRateLimit(30, 0))
	{
		authPublic.GET("/login", handleLogin(cfg, svc))
		authPublic.GET("/callback", handleCallback(cfg, svc))
		authPublic.POST("/logout", handleLogout(cfg))
	}

	r.GET("/api/shares/:code", handleAccessShare(svc))
	r.GET("/api/shares/:code/download", handleDownloadShare(svc))
	r.GET("/api/system/forbidden-image", handleForbiddenImage())

	// DogeCloud callback endpoint: support both GET and POST payload styles.
	callback := r.Group("/api/videos")
	callback.Use(middleware.StrictRateLimit(120, 0))
	{
		callback.GET("/callback", handleVideoCallback(cfg, svc))
		callback.POST("/callback", handleVideoCallback(cfg, svc))
		callback.GET("/callback/transcode", handleVideoCallback(cfg, svc))
		callback.POST("/callback/transcode", handleVideoCallback(cfg, svc))
	}

	// Direct link proxy (public, but file must exist)
	r.GET("/s/:user_id/:filename", handleDirectLink(svc))

	// Authenticated routes
	auth := r.Group("/api")
	auth.Use(middleware.Auth(cfg, svc.Auth.UserRepo()))

	// Session cookie for EventSource (cannot set Authorization headers)
	auth.POST("/auth/session", handleEstablishSession(cfg))

	// User info
	auth.GET("/user/me", handleGetMe())
	auth.GET("/user/quota", handleGetQuota())
	auth.GET("/user/history", handleGetHistory(svc))

	// Upload
	upload := auth.Group("/upload")
	upload.Use(middleware.QuotaCheck())
	upload.Use(middleware.StrictRateLimit(180, 0))
	{
		upload.POST("/init", handleInitUpload(svc))
		upload.POST("/:task_id/chunk", handleUploadChunk(svc))
		upload.PUT("/:task_id/pause", handlePauseUpload(svc))
		upload.PUT("/:task_id/resume", handleResumeUpload(svc))
		upload.POST("/:task_id/complete", handleCompleteUpload(svc))
		upload.GET("/tasks", handleListUploadTasks(svc))
	}

	// Files
	files := auth.Group("/files")
	{
		files.GET("/list", handleListFiles(svc))
		files.POST("/mkdir", handleMkdir(svc))
		files.PUT("/:id/rename", handleRename(svc))
		files.PUT("/:id/move", handleMove(svc))
		files.POST("/copy", handleCopy(svc))
		files.DELETE("/:id", handleSoftDelete(svc))
		files.DELETE("/batch", handleBatchDelete(svc))
		files.POST("/:id/restore", handleRestore(svc))
		files.DELETE("/:id/permanent", handlePermanentDelete(svc))
		files.GET("/trash", handleListTrash(svc))
		files.GET("/:id/preview", handlePreview(svc))
		files.GET("/:id/url", handleGetFileURL(svc))
		files.POST("/:id/share", handleCreateShare(svc))
		files.POST("/:id/appeal", handleSubmitFileAppeal(svc))
	}

	// Shares
	auth.GET("/shares", handleListShares(svc))
	auth.DELETE("/shares/:id", handleDeleteShare(svc))

	// Videos
	videos := auth.Group("/videos")
	{
		videos.POST("/upload", middleware.QuotaCheck(), handleVideoUpload(svc))
		videos.GET("/list", handleListVideos(svc))
		videos.PUT("/:id/status", handleSetVideoStatus(svc))
		videos.DELETE("/:id", handleDeleteVideo(svc))
		videos.DELETE("/batch", handleBatchDeleteVideos(svc))
		videos.GET("/:id/play", handleGetPlayURL(svc))
		videos.GET("/:id/play-info", handleGetPlayInfo(svc))
	}

	// SSE
	auth.GET("/events", handleSSE(svc))

	// Admin routes
	admin := auth.Group("/admin")
	admin.Use(middleware.RequireAdmin())
	{
		admin.GET("/files", handleAdminListFiles(svc))
		admin.PUT("/files/:id/ban", handleAdminBanFile(svc))
		admin.PUT("/files/:id/unban", handleAdminUnbanFile(svc))
		admin.DELETE("/files/:id", handleAdminDeleteFile(svc))
		admin.GET("/videos", handleAdminListVideos(svc))
		admin.GET("/videos/:id/play-info", handleAdminGetVideoPlayInfo(svc))
		admin.PUT("/videos/:id/status", handleAdminSetVideoStatus(svc))
		admin.DELETE("/videos/:id", handleAdminDeleteVideo(svc))
		admin.GET("/appeals", handleAdminListAppeals(svc))
		admin.PUT("/appeals/:id/review", handleAdminReviewAppeal(svc))
		admin.GET("/logs", handleAdminListLogs(svc))
		admin.GET("/users", handleAdminListUsers(svc))
		admin.PUT("/users/:id/quota", handleAdminUpdateQuota(svc))
		admin.POST("/users/recalc-storage", handleAdminRecalcStorage(svc))
		admin.DELETE("/trash/cleanup", handleAdminCleanupTrash(svc))
	}

	distDir := resolveFrontendDistDir()
	mountFrontendStatic(r, distDir)
	indexPath := filepath.Join(distDir, "index.html")
	if info, err := os.Stat(indexPath); err != nil || info.IsDir() {
		indexPath = resolveFrontendIndexPath()
	}

	// SPA fallback (after static assets)
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/s/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		// Avoid returning HTML for missing static-like paths
		if strings.HasPrefix(path, "/assets/") || strings.HasPrefix(path, "/vendor/") {
			c.Status(http.StatusNotFound)
			return
		}
		c.File(indexPath)
	})
}

