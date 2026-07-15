package handler

import (
	"errors"
	"log"
	"net/http"
	"strings"

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
	case isClientFacingError(err):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		log.Printf("request error path=%s err=%v", c.FullPath(), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败，请稍后重试"})
	}
}

func isClientFacingError(err error) bool {
	msg := err.Error()
	// Chinese business messages or short known English phrases are safe for clients.
	if strings.Contains(msg, "存储配额") ||
		strings.Contains(msg, "超过最大") ||
		strings.Contains(msg, "上传") ||
		strings.Contains(msg, "封禁") ||
		strings.Contains(msg, "申诉") ||
		strings.Contains(msg, "名称不能") ||
		strings.Contains(msg, "无效") ||
		strings.Contains(msg, "任务") ||
		strings.Contains(msg, "密码") ||
		strings.Contains(msg, "分享") {
		return true
	}
	return false
}
