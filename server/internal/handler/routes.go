package handler

import (
	"io"
	"net/http"
	"strconv"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/DTMWiki/IdeaSaver/server/internal/middleware"
	"github.com/DTMWiki/IdeaSaver/server/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SetupRoutes configures all HTTP routes.
func SetupRoutes(r *gin.Engine, cfg *config.Config, svc *service.Services) {
	// Public routes
	r.GET("/api/auth/login", handleLogin(svc))
	r.GET("/api/auth/callback", handleCallback(svc))
	r.GET("/api/shares/:code", handleAccessShare(svc))
	r.GET("/api/system/forbidden-image", handleForbiddenImage())
	r.POST("/api/videos/callback", handleVideoCallback(svc))

	// Direct link proxy (public, but file must exist)
	r.GET("/s/:user_id/:filename", handleDirectLink(svc))

	// Authenticated routes
	auth := r.Group("/api")
	auth.Use(middleware.Auth(cfg, svc.Auth.UserRepo()))

	// User info
	auth.GET("/user/me", handleGetMe())
	auth.GET("/user/quota", handleGetQuota())
	auth.GET("/user/history", handleGetHistory(svc))

	// Upload
	upload := auth.Group("/upload")
	upload.Use(middleware.QuotaCheck())
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
		videos.POST("/upload", handleVideoUpload(svc))
		videos.GET("/list", handleListVideos(svc))
		videos.PUT("/:id/status", handleSetVideoStatus(svc))
		videos.DELETE("/:id", handleDeleteVideo(svc))
		videos.DELETE("/batch", handleBatchDeleteVideos(svc))
		videos.GET("/:id/play", handleGetPlayURL(svc))
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
		admin.DELETE("/videos/:id", handleAdminDeleteVideo(svc))
		admin.GET("/appeals", handleAdminListAppeals(svc))
		admin.PUT("/appeals/:id/review", handleAdminReviewAppeal(svc))
		admin.GET("/logs", handleAdminListLogs(svc))
		admin.GET("/users", handleAdminListUsers(svc))
		admin.PUT("/users/:id/quota", handleAdminUpdateQuota(svc))
		admin.DELETE("/trash/cleanup", handleAdminCleanupTrash(svc))
	}

	// Serve frontend static files in production
	r.NoRoute(func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})
}

// --- Auth Handlers ---

func handleLogin(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		url := svc.Auth.GetAuthURL("state")
		c.JSON(http.StatusOK, gin.H{"url": url})
	}
}

func handleCallback(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Query("code")
		if code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "缺少授权码"})
			return
		}

		token, user, err := svc.Auth.ExchangeCode(c.Request.Context(), code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
	}
}

// --- User Handlers ---

func handleGetMe() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		c.JSON(http.StatusOK, gin.H{"user": user})
	}
}

func handleGetQuota() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		c.JSON(http.StatusOK, gin.H{
			"quota": user.StorageQuota,
			"used":  user.StorageUsed,
		})
	}
}

func handleGetHistory(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

		logs, total, err := svc.Admin.GetUserHistory(c.Request.Context(), user.ID, offset, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"logs": logs, "total": total})
	}
}

// --- Upload Handlers ---

func handleInitUpload(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		var req service.InitUploadRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		resp, err := svc.Upload.InitUpload(c.Request.Context(), user.ID, &req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func handleUploadChunk(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		taskID, err := uuid.Parse(c.Param("task_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务ID"})
			return
		}

		chunkIndex, _ := strconv.Atoi(c.PostForm("chunk_index"))
		file, header, err := c.Request.FormFile("chunk")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "缺少分片数据"})
			return
		}
		defer file.Close()

		if err := svc.Upload.UploadChunk(c.Request.Context(), taskID, user.ID, chunkIndex, file, header.Size); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handlePauseUpload(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		taskID, err := uuid.Parse(c.Param("task_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务ID"})
			return
		}
		if err := svc.Upload.PauseUpload(c.Request.Context(), taskID, user.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleResumeUpload(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		taskID, err := uuid.Parse(c.Param("task_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务ID"})
			return
		}
		if err := svc.Upload.ResumeUpload(c.Request.Context(), taskID, user.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleCompleteUpload(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		taskID, err := uuid.Parse(c.Param("task_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务ID"})
			return
		}

		var req struct {
			ParentID *uuid.UUID `json:"parent_id"`
		}
		_ = c.ShouldBindJSON(&req)

		file, err := svc.Upload.CompleteUpload(c.Request.Context(), taskID, user.ID, req.ParentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		url := file.PublicURL
		markdown := "[" + file.Name + "](" + url + ")"
		if isImageMime(file.MimeType) {
			markdown = "!" + markdown
		}

		c.JSON(http.StatusOK, gin.H{
			"file":     file,
			"url":      url,
			"markdown": markdown,
		})
	}
}

func handleListUploadTasks(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		tasks, err := svc.Upload.ListTasks(c.Request.Context(), user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"tasks": tasks})
	}
}

// --- File Handlers ---

func handleListFiles(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		var parentID *uuid.UUID
		if pid := c.Query("parent_id"); pid != "" {
			id, err := uuid.Parse(pid)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的父目录ID"})
				return
			}
			parentID = &id
		}

		files, err := svc.File.ListFiles(c.Request.Context(), user.ID, parentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"files": files})
	}
}

func handleMkdir(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		var req struct {
			Name     string     `json:"name"`
			ParentID *uuid.UUID `json:"parent_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		dir, err := svc.File.CreateDirectory(c.Request.Context(), user.ID, req.ParentID, req.Name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"directory": dir})
	}
}

func handleRename(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件ID"})
			return
		}
		var req struct {
			Name string `json:"name"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := svc.File.Rename(c.Request.Context(), id, user.ID, req.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleMove(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件ID"})
			return
		}
		var req struct {
			ParentID *uuid.UUID `json:"parent_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := svc.File.Move(c.Request.Context(), id, user.ID, req.ParentID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleCopy(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		var req struct {
			FileID   uuid.UUID  `json:"file_id"`
			ParentID *uuid.UUID `json:"parent_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		file, err := svc.File.Copy(c.Request.Context(), req.FileID, user.ID, req.ParentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"file": file})
	}
}

func handleSoftDelete(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件ID"})
			return
		}
		if err := svc.File.SoftDelete(c.Request.Context(), id, user.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleBatchDelete(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		var req struct {
			IDs []uuid.UUID `json:"ids"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		for _, id := range req.IDs {
			_ = svc.File.SoftDelete(c.Request.Context(), id, user.ID)
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleRestore(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件ID"})
			return
		}
		if err := svc.File.Restore(c.Request.Context(), id, user.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handlePermanentDelete(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件ID"})
			return
		}
		if err := svc.File.PermanentDelete(c.Request.Context(), id, user.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleListTrash(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		files, err := svc.File.ListTrash(c.Request.Context(), user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"files": files})
	}
}

func handlePreview(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件ID"})
			return
		}

		file, err := svc.File.GetFileByID(c.Request.Context(), id, user.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
			return
		}
		if file.ModerationStatus == "banned" {
			c.Redirect(http.StatusFound, "/api/system/forbidden-image")
			return
		}

		reader, contentType, contentLength, err := svc.File.ProxyFile(c.Request.Context(), file.StorageKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer reader.Close()

		c.Header("Content-Type", contentType)
		if contentLength > 0 {
			c.Header("Content-Length", strconv.FormatInt(contentLength, 10))
		}
		io.Copy(c.Writer, reader)
	}
}

func handleGetFileURL(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件ID"})
			return
		}

		url, markdown, err := svc.File.GetFileURL(c.Request.Context(), id, user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"url": url, "markdown": markdown})
	}
}

func handleSubmitFileAppeal(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件ID"})
			return
		}

		var req struct {
			Reason string `json:"reason"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		appeal, err := svc.File.SubmitAppeal(c.Request.Context(), id, user.ID, req.Reason)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"appeal": appeal})
	}
}

// --- Share Handlers ---

func handleCreateShare(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件ID"})
			return
		}
		var req struct {
			Password  string `json:"password"`
			ExpiresIn int    `json:"expires_in"`
		}
		_ = c.ShouldBindJSON(&req)

		share, err := svc.Share.CreateShare(c.Request.Context(), user.ID, &service.CreateShareRequest{
			FileID:    id,
			Password:  req.Password,
			ExpiresIn: req.ExpiresIn,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"share": share})
	}
}

func handleAccessShare(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Param("code")
		password := c.Query("password")

		file, err := svc.Share.AccessShare(c.Request.Context(), code, password)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"file": file})
	}
}

func handleListShares(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		shares, err := svc.Share.ListShares(c.Request.Context(), user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"shares": shares})
	}
}

func handleDeleteShare(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的分享ID"})
			return
		}
		if err := svc.Share.DeleteShare(c.Request.Context(), id, user.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// --- Video Handlers ---

func handleVideoUpload(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		title := c.PostForm("title")
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "缺少视频文件"})
			return
		}
		defer file.Close()

		if title == "" {
			title = header.Filename
		}

		video, err := svc.Video.UploadVideo(c.Request.Context(), user.ID, title, file, header.Filename, header.Size)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"video": video})
	}
}

func handleListVideos(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

		videos, total, err := svc.Video.ListVideos(c.Request.Context(), user.ID, offset, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"videos": videos, "total": total})
	}
}

func handleSetVideoStatus(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的视频ID"})
			return
		}
		var req struct {
			Status int16 `json:"status"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := svc.Video.SetVideoStatus(c.Request.Context(), id, user.ID, req.Status); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleDeleteVideo(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的视频ID"})
			return
		}
		if err := svc.Video.DeleteVideo(c.Request.Context(), id, user.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleBatchDeleteVideos(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		var req struct {
			IDs []uuid.UUID `json:"ids"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := svc.Video.BatchDeleteVideos(c.Request.Context(), req.IDs, user.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleGetPlayURL(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的视频ID"})
			return
		}

		playURL, err := svc.Video.GetPlayURL(c.Request.Context(), id, user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"play_url": playURL})
	}
}

func handleVideoCallback(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		msg := c.Query("msg")
		vid := c.Query("vid")
		callback := c.Query("callback")

		if err := svc.Video.HandleCallback(c.Request.Context(), msg, vid, callback); err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		c.String(http.StatusOK, "DogeCloud Callback Success")
	}
}

// --- Direct Link Proxy ---

func handleDirectLink(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("user_id")
		filename := c.Param("filename")
		if _, err := uuid.Parse(userID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
			return
		}

		file, err := svc.File.ResolvePublicFile(c.Request.Context(), userID, filename)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
			return
		}
		if file.ModerationStatus == "banned" {
			c.Redirect(http.StatusFound, "/api/system/forbidden-image")
			return
		}

		reader, contentType, contentLength, err := svc.File.ProxyFile(c.Request.Context(), file.StorageKey)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
			return
		}
		defer reader.Close()

		c.Header("Content-Type", contentType)
		if contentLength > 0 {
			c.Header("Content-Length", strconv.FormatInt(contentLength, 10))
		}
		c.Header("Cache-Control", "public, max-age=86400")
		io.Copy(c.Writer, reader)
	}
}

func handleForbiddenImage() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Data(http.StatusOK, "image/svg+xml", []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630" viewBox="0 0 1200 630"><rect width="1200" height="630" fill="#fff5f5"/><rect x="40" y="40" width="1120" height="550" rx="24" fill="#fff" stroke="#ffc9c9" stroke-width="4"/><circle cx="180" cy="180" r="52" fill="#ff6b6b"/><path d="M180 148 L180 188" stroke="#fff" stroke-width="10" stroke-linecap="round"/><circle cx="180" cy="218" r="6" fill="#fff"/><text x="280" y="190" fill="#c92a2a" font-size="64" font-family="Arial, sans-serif" font-weight="700">Access Denied</text><text x="280" y="250" fill="#495057" font-size="32" font-family="Arial, sans-serif">This resource is under review by the platform administrator.</text><text x="280" y="310" fill="#868e96" font-size="24" font-family="Arial, sans-serif">If you are the owner, please submit an appeal ticket in the dashboard.</text></svg>`))
	}
}

// --- SSE Handler ---

func handleSSE(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)

		client := &service.SSEClient{
			ID:     uuid.New().String(),
			UserID: user.ID,
			Events: make(chan service.SSEEvent, 10),
		}

		svc.SSE.Register(client)
		defer svc.SSE.Unregister(client.ID)

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no") // Disable Nginx buffering for SSE

		c.Writer.Flush()

		for {
			select {
			case event, ok := <-client.Events:
				if !ok {
					return
				}
				c.Writer.WriteString(service.FormatSSE(event))
				c.Writer.Flush()
			case <-c.Request.Context().Done():
				return
			}
		}
	}
}

// --- Admin Handlers ---

func handleAdminListFiles(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

		files, total, err := svc.Admin.ListAllFiles(c.Request.Context(), offset, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"files": files, "total": total})
	}
}

func handleAdminDeleteFile(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件ID"})
			return
		}
		if err := svc.Admin.DeleteFile(c.Request.Context(), id, admin.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleAdminBanFile(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件ID"})
			return
		}

		var req struct {
			Reason string `json:"reason"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := svc.Admin.BanFile(c.Request.Context(), id, admin.ID, req.Reason); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleAdminUnbanFile(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件ID"})
			return
		}

		var req struct {
			Comment string `json:"comment"`
		}
		_ = c.ShouldBindJSON(&req)

		if err := svc.Admin.UnbanFile(c.Request.Context(), id, admin.ID, req.Comment); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleAdminListVideos(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

		videos, total, err := svc.Admin.ListAllVideos(c.Request.Context(), offset, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"videos": videos, "total": total})
	}
}

func handleAdminDeleteVideo(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的视频ID"})
			return
		}
		if err := svc.Admin.DeleteVideo(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleAdminListAppeals(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.Query("status")
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

		appeals, total, err := svc.Admin.ListAppeals(c.Request.Context(), status, offset, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"appeals": appeals, "total": total})
	}
}

func handleAdminReviewAppeal(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的工单ID"})
			return
		}

		var req struct {
			Decision string `json:"decision"`
			Comment  string `json:"comment"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := svc.Admin.ReviewAppeal(c.Request.Context(), id, admin.ID, req.Decision, req.Comment); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleAdminListLogs(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		action := c.Query("action")
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

		logs, total, err := svc.Admin.ListAuditLogs(c.Request.Context(), action, offset, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"logs": logs, "total": total})
	}
}

func handleAdminListUsers(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

		users, total, err := svc.Admin.ListUsers(c.Request.Context(), offset, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"users": users, "total": total})
	}
}

func handleAdminUpdateQuota(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
			return
		}
		var req struct {
			Quota int64 `json:"quota"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := svc.Admin.UpdateUserQuota(c.Request.Context(), id, req.Quota); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleAdminCleanupTrash(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		count, err := svc.Admin.CleanupTrash(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"cleaned": count})
	}
}

// --- Helpers ---

func isImageMime(mime string) bool {
	return len(mime) > 6 && mime[:6] == "image/"
}
