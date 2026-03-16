package middleware

import (
	"github.com/DTMWiki/IdeaSaver/server/internal/model"
	"github.com/DTMWiki/IdeaSaver/server/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Audit logs all write operations to the audit_logs table.
func Audit(auditRepo *repository.AuditLogRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Only log write operations
		if c.Request.Method == "GET" || c.Request.Method == "OPTIONS" || c.Request.Method == "HEAD" {
			return
		}

		// Skip if response was an error (4xx/5xx will still be logged for security)
		user := GetUser(c)
		if user == nil {
			return
		}

		action := inferAction(c)
		if action == "" {
			return
		}

		log := &model.AuditLog{
			UserID:    user.ID,
			Action:    action,
			Resource:  inferResource(c),
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
		}

		// Try to get resource ID from URL params
		if idStr := c.Param("id"); idStr != "" {
			if id, err := uuid.Parse(idStr); err == nil {
				log.ResourceID = &id
			}
		}

		_ = auditRepo.Create(c.Request.Context(), log)
	}
}

// inferAction determines the action from the request method and path.
func inferAction(c *gin.Context) string {
	path := c.FullPath()
	method := c.Request.Method

	switch {
	case contains(path, "/upload") && method == "POST":
		return "upload"
	case contains(path, "/mkdir") && method == "POST":
		return "create_directory"
	case contains(path, "/rename") && method == "PUT":
		return "rename"
	case contains(path, "/move") && method == "PUT":
		return "move"
	case contains(path, "/copy") && method == "POST":
		return "copy"
	case contains(path, "/restore") && method == "POST":
		return "restore"
	case contains(path, "/share") && method == "POST":
		return "share"
	case contains(path, "/permanent") && method == "DELETE":
		return "permanent_delete"
	case method == "DELETE":
		return "delete"
	case contains(path, "/status") && method == "PUT":
		return "status_change"
	case contains(path, "/quota") && method == "PUT":
		return "quota_change"
	default:
		return method + " " + path
	}
}

// inferResource determines the resource type from the request path.
func inferResource(c *gin.Context) string {
	path := c.FullPath()
	switch {
	case contains(path, "/files"):
		return "file"
	case contains(path, "/videos"):
		return "video"
	case contains(path, "/shares"):
		return "share"
	case contains(path, "/users"):
		return "user"
	case contains(path, "/upload"):
		return "upload"
	default:
		return "unknown"
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
