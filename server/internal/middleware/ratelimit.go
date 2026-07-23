package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/gin-gonic/gin"
)

type visitor struct {
	count    int
	windowAt time.Time
}

// ipRateLimiter is a simple fixed-window limiter keyed by client IP.
type ipRateLimiter struct {
	mu      sync.Mutex
	visitors map[string]*visitor
	limit   int
	window  time.Duration
}

func newIPRateLimiter(limit int, window time.Duration) *ipRateLimiter {
	rl := &ipRateLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
		window:   window,
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *ipRateLimiter) allow(key string) bool {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, ok := rl.visitors[key]
	if !ok || now.Sub(v.windowAt) >= rl.window {
		rl.visitors[key] = &visitor{count: 1, windowAt: now}
		return true
	}
	if v.count >= rl.limit {
		return false
	}
	v.count++
	return true
}

func (rl *ipRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		rl.mu.Lock()
		for k, v := range rl.visitors {
			if now.Sub(v.windowAt) > rl.window*2 {
				delete(rl.visitors, k)
			}
		}
		// Hard cap to bound memory under IP-flood.
		if len(rl.visitors) > 100_000 {
			rl.visitors = make(map[string]*visitor)
		}
		rl.mu.Unlock()
	}
}

// RateLimit applies a global per-IP request budget (default: 300/min).
// Note: MaxConcurrentUploads / UploadRateLimitMBps are enforced in UploadService, not here.
func RateLimit(cfg *config.Config) gin.HandlerFunc {
	_ = cfg
	limit := 300
	rl := newIPRateLimiter(limit, time.Minute)
	return func(c *gin.Context) {
		if !rl.allow(c.ClientIP()) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "请求过于频繁，请稍后再试"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// StrictRateLimit is for sensitive endpoints (login, callbacks, uploads).
func StrictRateLimit(limit int, window time.Duration) gin.HandlerFunc {
	if limit <= 0 {
		limit = 60
	}
	if window <= 0 {
		window = time.Minute
	}
	rl := newIPRateLimiter(limit, window)
	return func(c *gin.Context) {
		if !rl.allow(c.ClientIP() + "|" + c.FullPath()) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "请求过于频繁，请稍后再试"})
			c.Abort()
			return
		}
		c.Next()
	}
}
