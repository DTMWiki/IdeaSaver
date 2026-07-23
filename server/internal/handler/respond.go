package handler

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"unicode"

	"github.com/DTMWiki/IdeaSaver/server/internal/service"
	"github.com/gin-gonic/gin"
)

// writeServiceError maps service-layer errors to safe HTTP responses.
func writeServiceError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	switch {
	case errors.Is(err, service.ErrPermission):
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限"})
	case errors.Is(err, service.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "资源不存在"})
	default:
		var ce *service.ClientError
		if errors.As(err, &ce) {
			c.JSON(http.StatusBadRequest, gin.H{"error": ce.Msg})
			return
		}
		if isSafeBusinessMessage(err.Error()) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.Printf("request error path=%s err=%v", c.FullPath(), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败，请稍后重试"})
	}
}

// isSafeBusinessMessage allows short Chinese product messages through, but blocks
// English/stack-like internals (avoids substring false positives like "分享" in stack traces).
func isSafeBusinessMessage(msg string) bool {
	msg = strings.TrimSpace(msg)
	if msg == "" || len(msg) > 160 {
		return false
	}
	lower := strings.ToLower(msg)
	for _, bad := range []string{"failed to", "sql:", "panic", "stack", "runtime.", "pq:", "http:", "json:"} {
		if strings.Contains(lower, bad) {
			return false
		}
	}
	hasCJK := false
	for _, r := range msg {
		if unicode.Is(unicode.Han, r) {
			hasCJK = true
			break
		}
	}
	return hasCJK
}
