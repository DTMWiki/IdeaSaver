package middleware

import (
	"github.com/DTMWiki/IdeaSaver/server/internal/requestctx"
	"github.com/gin-gonic/gin"
)

func RequestMeta() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := requestctx.WithMeta(c.Request.Context(), c.ClientIP(), c.Request.UserAgent())
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
