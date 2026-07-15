package handler

import (
	"crypto/subtle"
	"net/http"
	"strconv"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/DTMWiki/IdeaSaver/server/internal/middleware"
	"github.com/DTMWiki/IdeaSaver/server/internal/service"
	"github.com/gin-gonic/gin"
)

func handleLogin(cfg *config.Config, svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Random OAuth state is generated inside AuthService.BeginAuth (crypto/rand).
		url, state, err := svc.Auth.BeginAuth()
		if err != nil || url == "" || state == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法发起登录"})
			return
		}
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(oauthStateCookie, state, 600, "/", "", cookieSecure(cfg), true)
		c.JSON(http.StatusOK, gin.H{"url": url})
	}
}

func handleCallback(cfg *config.Config, svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Query("code")
		if code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "缺少授权码"})
			return
		}

		state := c.Query("state")
		cookieState, err := c.Cookie(oauthStateCookie)
		if err != nil || state == "" || cookieState == "" ||
			subtle.ConstantTimeCompare([]byte(state), []byte(cookieState)) != 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "登录状态无效，请重新登录"})
			return
		}
		// Clear one-time state cookie
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(oauthStateCookie, "", -1, "/", "", cookieSecure(cfg), true)

		token, user, err := svc.Auth.ExchangeCode(c.Request.Context(), code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "登录失败，请重试"})
			return
		}

		// Session is cookie-only for browsers; do not return JWT in JSON body.
		setAuthTokenCookie(c, cfg, token)
		c.JSON(http.StatusOK, gin.H{"user": user})
	}
}

// handleSessionMe is an explicit session probe (same as /user/me, public path under auth group).
func handleEstablishSession(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Auth middleware already validated cookie/Bearer; refresh cookie expiry if Bearer was used.
		if token := extractBearerToken(c); token != "" {
			setAuthTokenCookie(c, cfg, token)
		}
		user := middleware.GetUser(c)
		c.JSON(http.StatusOK, gin.H{"ok": true, "user": user})
	}
}

func handleLogout(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		clearAuthTokenCookie(c, cfg)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func setAuthTokenCookie(c *gin.Context, cfg *config.Config, token string) {
	// Match JWT lifetime (24h) used by AuthService.generateJWT.
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(authTokenCookie, token, 24*60*60, "/", "", cookieSecure(cfg), true)
}

func clearAuthTokenCookie(c *gin.Context, cfg *config.Config) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(authTokenCookie, "", -1, "/", "", cookieSecure(cfg), true)
}

func extractBearerToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}
	const prefix = "Bearer "
	if len(authHeader) <= len(prefix) || authHeader[:len(prefix)] != prefix {
		return ""
	}
	return authHeader[len(prefix):]
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
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"logs": logs, "total": total})
	}
}
