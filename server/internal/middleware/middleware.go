package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/DTMWiki/IdeaSaver/server/internal/model"
	"github.com/DTMWiki/IdeaSaver/server/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const UserContextKey contextKey = "user"

// Claims represents JWT token claims.
type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Role     string    `json:"role"`
	jwt.RegisteredClaims
}

// AuthTokenCookie is the HttpOnly cookie used for SSE and same-origin browser auth.
const AuthTokenCookie = "ideasaver_token"

// Auth verifies JWT token and injects user info into context.
// Token resolution order: Authorization Bearer → HttpOnly cookie.
// Query-string tokens are intentionally unsupported (log/referrer leakage).
func Auth(cfg *config.Config, userRepo *repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := ""

		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr == authHeader {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的认证格式"})
				c.Abort()
				return
			}
		}

		if tokenStr == "" {
			if cookie, err := c.Cookie(AuthTokenCookie); err == nil {
				tokenStr = strings.TrimSpace(cookie)
			}
		}

		if tokenStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录，请先登录"})
			c.Abort()
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "认证已过期或无效"})
			c.Abort()
			return
		}

		// Fetch full user from DB
		user, err := userRepo.FindByID(c.Request.Context(), claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
			c.Abort()
			return
		}

		c.Set(string(UserContextKey), user)
		c.Next()
	}
}

// RequireAdmin checks if the authenticated user has admin role.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetUser(c)
		if user == nil || user.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// GetUser extracts the authenticated user from Gin context.
func GetUser(c *gin.Context) *model.User {
	val, exists := c.Get(string(UserContextKey))
	if !exists {
		return nil
	}
	user, ok := val.(*model.User)
	if !ok {
		return nil
	}
	return user
}

// GetUserFromContext extracts user from standard context (used in services).
func GetUserFromContext(ctx context.Context) *model.User {
	val := ctx.Value(UserContextKey)
	if val == nil {
		return nil
	}
	user, ok := val.(*model.User)
	if !ok {
		return nil
	}
	return user
}

// CORS configures Cross-Origin Resource Sharing headers.
func CORS(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := cfg.PublicBaseURL
		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// QuotaCheck verifies the user has enough storage quota before upload.
func QuotaCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetUser(c)
		if user == nil {
			c.Next()
			return
		}

		if user.StorageUsed >= user.StorageQuota {
			c.JSON(http.StatusForbidden, gin.H{"error": "存储配额已用尽，请联系管理员"})
			c.Abort()
			return
		}

		c.Next()
	}
}
