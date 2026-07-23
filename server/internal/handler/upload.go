package handler

import (
	"net/http"
	"strconv"

	"github.com/DTMWiki/IdeaSaver/server/internal/middleware"
	"github.com/DTMWiki/IdeaSaver/server/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func handleInitUpload(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		var req service.InitUploadRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
			return
		}

		resp, err := svc.Upload.InitUpload(c.Request.Context(), user.ID, &req)
		if err != nil {
			writeServiceError(c, err)
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

		chunkIndex, err := strconv.Atoi(c.PostForm("chunk_index"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的分片索引"})
			return
		}
		file, header, err := c.Request.FormFile("chunk")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "缺少分片数据"})
			return
		}
		defer file.Close()

		if err := svc.Upload.UploadChunk(c.Request.Context(), taskID, user.ID, chunkIndex, file, header.Size); err != nil {
			writeServiceError(c, err)
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
			writeServiceError(c, err)
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
			writeServiceError(c, err)
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
			writeServiceError(c, err)
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
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"tasks": tasks})
	}
}

// --- File Handlers ---

