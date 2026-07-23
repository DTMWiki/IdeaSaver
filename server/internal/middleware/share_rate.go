package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SharePasswordLimit rate-limits share access/download failures per IP+code.
// Successful unlocks are cheap; failed password attempts are counted.
func SharePasswordLimit(maxFails int, window time.Duration) gin.HandlerFunc {
	if maxFails <= 0 {
		maxFails = 10
	}
	if window <= 0 {
		window = time.Minute
	}
	type bucket struct {
		fails    int
		windowAt time.Time
	}
	var mu sync.Mutex
	visitors := map[string]*bucket{}

	go func() {
		t := time.NewTicker(5 * time.Minute)
		for range t.C {
			now := time.Now()
			mu.Lock()
			for k, v := range visitors {
				if now.Sub(v.windowAt) > window*2 {
					delete(visitors, k)
				}
			}
			// Soft cap map size
			if len(visitors) > 50_000 {
				visitors = map[string]*bucket{}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		code := c.Param("code")
		key := c.ClientIP() + "|" + code
		mu.Lock()
		b, ok := visitors[key]
		now := time.Now()
		if !ok || now.Sub(b.windowAt) >= window {
			b = &bucket{fails: 0, windowAt: now}
			visitors[key] = b
		}
		blocked := b.fails >= maxFails
		mu.Unlock()

		if blocked {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "尝试次数过多，请稍后再试",
				"code":  "share_rate_limited",
			})
			c.Abort()
			return
		}

		c.Next()

		// Count failed password-related responses.
		status := c.Writer.Status()
		if status == http.StatusForbidden {
			// Only count when password was involved (body code set by handlers via context).
			if v, exists := c.Get("share_auth_failed"); exists && v == true {
				mu.Lock()
				if b2, ok := visitors[key]; ok {
					if time.Since(b2.windowAt) >= window {
						b2.fails = 1
						b2.windowAt = time.Now()
					} else {
						b2.fails++
					}
				}
				mu.Unlock()
			}
		}
		if status == http.StatusOK {
			mu.Lock()
			delete(visitors, key)
			mu.Unlock()
		}
	}
}

// MarkShareAuthFailed records that this request failed share password auth.
func MarkShareAuthFailed(c *gin.Context) {
	c.Set("share_auth_failed", true)
}
