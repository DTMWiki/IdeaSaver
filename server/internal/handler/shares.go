package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/DTMWiki/IdeaSaver/server/internal/middleware"
	"github.com/DTMWiki/IdeaSaver/server/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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
			writeServiceError(c, err)
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
			writeShareAccessError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"file": file})
	}
}

func handleDownloadShare(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Param("code")
		password := c.Query("password")

		reader, contentType, contentLength, view, err := svc.Share.ProxySharedFile(c.Request.Context(), code, password)
		if err != nil {
			writeShareAccessError(c, err)
			return
		}
		defer reader.Close()

		if contentType != "" {
			c.Header("Content-Type", contentType)
		}
		if contentLength > 0 {
			c.Header("Content-Length", strconv.FormatInt(contentLength, 10))
		}
		if view != nil && view.Name != "" {
			c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%q", view.Name))
		}
		c.Header("Cache-Control", "private, no-store")
		io.Copy(c.Writer, reader)
	}
}

func writeShareAccessError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrSharePasswordRequired):
		c.JSON(http.StatusForbidden, gin.H{"error": "需要密码", "code": "password_required"})
	case errors.Is(err, service.ErrSharePasswordInvalid):
		c.JSON(http.StatusForbidden, gin.H{"error": "密码错误", "code": "password_invalid"})
	case errors.Is(err, service.ErrShareExpired):
		c.JSON(http.StatusForbidden, gin.H{"error": "分享链接已过期", "code": "share_expired"})
	case errors.Is(err, service.ErrShareBanned):
		c.JSON(http.StatusForbidden, gin.H{"error": "该分享资源已被封禁", "code": "share_banned"})
	case errors.Is(err, service.ErrShareNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "分享链接不存在", "code": "share_not_found"})
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "无法访问该分享", "code": "share_access_denied"})
	}
}

func handleListShares(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		shares, err := svc.Share.ListShares(c.Request.Context(), user.ID)
		if err != nil {
			writeServiceError(c, err)
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
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// --- Video Handlers ---

