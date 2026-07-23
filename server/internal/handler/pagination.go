package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func parseOffsetLimit(c *gin.Context, defaultLimit, maxLimit int) (offset, limit int) {
	if defaultLimit <= 0 {
		defaultLimit = 50
	}
	if maxLimit <= 0 {
		maxLimit = 200
	}
	offset, _ = strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ = strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultLimit)))
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return offset, limit
}
