package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders sets baseline browser security headers for all responses.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		// Tight default CSP; allow Doge player CDN used by the video page.
		h.Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline' https://player.dogecloud.com; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: blob: https:; "+
				"media-src 'self' blob: https:; "+
				"frame-src 'self' https://player.dogecloud.com; "+
				"connect-src 'self' https://player.dogecloud.com; "+
				"object-src 'none'; "+
				"base-uri 'self'; "+
				"frame-ancestors 'self'")
		// HSTS only meaningful on HTTPS; safe to set (browsers ignore on plain HTTP).
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Next()
	}
}
