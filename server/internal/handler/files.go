package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/DTMWiki/IdeaSaver/server/internal/middleware"
	"github.com/DTMWiki/IdeaSaver/server/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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
			writeServiceError(c, err)
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
			return
		}

		dir, err := svc.File.CreateDirectory(c.Request.Context(), user.ID, req.ParentID, req.Name)
		if err != nil {
			writeServiceError(c, err)
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
			return
		}

		if err := svc.File.Rename(c.Request.Context(), id, user.ID, req.Name); err != nil {
			writeServiceError(c, err)
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
			return
		}

		if err := svc.File.Move(c.Request.Context(), id, user.ID, req.ParentID); err != nil {
			writeServiceError(c, err)
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
			return
		}

		file, err := svc.File.Copy(c.Request.Context(), req.FileID, user.ID, req.ParentID)
		if err != nil {
			writeServiceError(c, err)
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
			writeServiceError(c, err)
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
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
			writeServiceError(c, err)
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
			writeServiceError(c, err)
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
			writeServiceError(c, err)
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

		file, err := svc.File.GetFileForActor(c.Request.Context(), id, user)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
			return
		}
		if file.ModerationStatus == "banned" {
			c.Redirect(http.StatusFound, "/api/system/forbidden-image")
			return
		}

		thumbMode := c.Query("thumb") == "1"
		var (
			reader        io.ReadCloser
			contentType   string
			contentLength int64
		)
		if thumbMode {
			if !strings.HasPrefix(strings.TrimSpace(file.MimeType), "image/") {
				c.JSON(http.StatusBadRequest, gin.H{"error": "该文件不支持缩略图"})
				return
			}
			reader, contentType, contentLength, err = svc.File.ProxyThumbnail(c.Request.Context(), file.StorageKey, "thumb")
		} else {
			reader, contentType, contentLength, err = svc.File.ProxyFile(c.Request.Context(), file.StorageKey)
		}
		if err != nil {
			writeServiceError(c, err)
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

		url, markdown, err := svc.File.GetFileURLForActor(c.Request.Context(), id, user)
		if err != nil {
			writeServiceError(c, err)
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
			return
		}

		appeal, err := svc.File.SubmitAppeal(c.Request.Context(), id, user.ID, req.Reason)
		if err != nil {
			writeServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"appeal": appeal})
	}
}

// --- Share Handlers ---

