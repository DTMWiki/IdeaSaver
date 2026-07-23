package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/DTMWiki/IdeaSaver/server/internal/middleware"
	"github.com/DTMWiki/IdeaSaver/server/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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

		// Never render HTML/SVG/JS inline on the app origin (stored XSS).
		serveType := safeServeContentType(contentType, file.Name)
		c.Header("Content-Type", serveType)
		c.Header("X-Content-Type-Options", "nosniff")
		if forceAttachment(contentType, file.Name) || serveType == "application/octet-stream" {
			c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", file.Name))
		} else {
			c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%q", file.Name))
		}
		if contentLength > 0 {
			c.Header("Content-Length", strconv.FormatInt(contentLength, 10))
		}
		// Short private cache: banned/deleted resources must not stick in CDN for a day.
		c.Header("Cache-Control", "private, max-age=300")
		io.Copy(c.Writer, reader)
	}
}

func handleForbiddenImage() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Data(http.StatusOK, "image/svg+xml", []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630" viewBox="0 0 1200 630"><rect width="1200" height="630" fill="#fff5f5"/><rect x="40" y="40" width="1120" height="550" rx="24" fill="#fff" stroke="#ffc9c9" stroke-width="4"/><circle cx="180" cy="180" r="52" fill="#ff6b6b"/><path d="M180 148 L180 188" stroke="#fff" stroke-width="10" stroke-linecap="round"/><circle cx="180" cy="218" r="6" fill="#fff"/><text x="280" y="190" fill="#c92a2a" font-size="56" font-family="Microsoft YaHei, PingFang SC, Arial, sans-serif" font-weight="700">无法访问</text><text x="280" y="250" fill="#495057" font-size="28" font-family="Microsoft YaHei, PingFang SC, Arial, sans-serif">该资源已被管理员限制访问。</text><text x="280" y="310" fill="#868e96" font-size="22" font-family="Microsoft YaHei, PingFang SC, Arial, sans-serif">如果你是文件所有者，请在控制台提交申诉。</text></svg>`))
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

		if !svc.SSE.Register(client) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "SSE 连接数已达上限，请关闭其他标签页后重试"})
			return
		}
		defer svc.SSE.Unregister(client.ID)

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache, no-transform")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no") // Disable Nginx buffering for SSE
		c.Status(http.StatusOK)
		c.Writer.WriteHeaderNow()

		if _, err := c.Writer.WriteString("retry: 5000\n: connected\n\n"); err != nil {
			return
		}
		c.Writer.Flush()

		heartbeat := time.NewTicker(25 * time.Second)
		defer heartbeat.Stop()

		for {
			select {
			case event, ok := <-client.Events:
				if !ok {
					return
				}
				if _, err := c.Writer.WriteString(service.FormatSSE(event)); err != nil {
					return
				}
				c.Writer.Flush()
			case <-heartbeat.C:
				if _, err := c.Writer.WriteString(": ping\n\n"); err != nil {
					return
				}
				c.Writer.Flush()
			case <-c.Request.Context().Done():
				return
			}
		}
	}
}

// --- Admin Handlers ---

