package handler

import (
	"crypto/subtle"
	"net/http"
	"strconv"
	"strings"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/DTMWiki/IdeaSaver/server/internal/middleware"
	"github.com/DTMWiki/IdeaSaver/server/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func handleVideoUpload(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		title := c.PostForm("title")
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "缺少视频文件"})
			return
		}
		defer file.Close()

		if title == "" {
			title = header.Filename
		}

		video, err := svc.Video.UploadVideo(c.Request.Context(), user.ID, title, file, header.Filename, header.Size)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"video": video})
	}
}

func handleListVideos(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		offset, limit := parseOffsetLimit(c, 20, 200)

		videos, total, err := svc.Video.ListVideos(c.Request.Context(), user.ID, offset, limit)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"videos": videos, "total": total})
	}
}

func handleSetVideoStatus(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的视频ID"})
			return
		}
		var req struct {
			Status int16 `json:"status"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
			return
		}

		if err := svc.Video.SetVideoStatus(c.Request.Context(), id, user.ID, req.Status); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleDeleteVideo(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的视频ID"})
			return
		}
		if err := svc.Video.DeleteVideo(c.Request.Context(), id, user.ID); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleBatchDeleteVideos(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		var req struct {
			IDs []uuid.UUID `json:"ids"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
			return
		}

		if err := svc.Video.BatchDeleteVideos(c.Request.Context(), req.IDs, user.ID); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleGetPlayURL(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的视频ID"})
			return
		}

		info, err := svc.Video.GetPlayInfoForViewer(c.Request.Context(), id, user.ID, c.ClientIP(), c.Request.UserAgent())
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, info)
	}
}

func handleGetPlayInfo(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的视频ID"})
			return
		}

		info, err := svc.Video.GetPlayInfoForViewer(c.Request.Context(), id, user.ID, c.ClientIP(), c.Request.UserAgent())
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, info)
	}
}

func handleVideoCallback(cfg *config.Config, svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !authorizeVideoCallback(cfg, c) {
			c.String(http.StatusUnauthorized, "unauthorized callback")
			return
		}

		payload := service.VideoCallbackPayload{
			Msg:          strings.TrimSpace(c.Query("msg")),
			VID:          strings.TrimSpace(c.Query("vid")),
			VCode:        strings.TrimSpace(c.Query("vcode")),
			PlayerUserID: strings.TrimSpace(firstNonEmpty(c.Query("userId"), c.Query("userid"))),
			Callback:     strings.TrimSpace(firstNonEmpty(c.Query("callbackString"), c.Query("callback"))),
		}

		if c.Request.Method == http.MethodPost {
			_ = c.Request.ParseForm()
			payload.Msg = strings.TrimSpace(firstNonEmpty(payload.Msg, c.PostForm("msg")))
			payload.VID = strings.TrimSpace(firstNonEmpty(payload.VID, c.PostForm("vid")))
			payload.VCode = strings.TrimSpace(firstNonEmpty(payload.VCode, c.PostForm("vcode")))
			payload.PlayerUserID = strings.TrimSpace(firstNonEmpty(
				payload.PlayerUserID,
				c.PostForm("userId"),
				c.PostForm("userid"),
			))
			payload.Callback = strings.TrimSpace(firstNonEmpty(
				payload.Callback,
				c.PostForm("callbackString"),
				c.PostForm("callback"),
			))

			var body map[string]any
			if err := c.ShouldBindJSON(&body); err == nil {
				payload.Msg = strings.TrimSpace(firstNonEmpty(payload.Msg, asString(body["msg"])))
				payload.VID = strings.TrimSpace(firstNonEmpty(payload.VID, asString(body["vid"])))
				payload.VCode = strings.TrimSpace(firstNonEmpty(payload.VCode, asString(body["vcode"])))
				payload.PlayerUserID = strings.TrimSpace(firstNonEmpty(
					payload.PlayerUserID,
					asString(body["userId"]),
					asString(body["userid"]),
				))
				payload.Callback = strings.TrimSpace(firstNonEmpty(
					payload.Callback,
					asString(body["callbackString"]),
					asString(body["callback"]),
				))
			}
		}

		if err := svc.Video.HandleCallback(c.Request.Context(), payload); err != nil {
			c.String(http.StatusInternalServerError, "callback failed")
			return
		}
		c.String(http.StatusOK, "DogeCloud Callback Success")
	}
}

func authorizeVideoCallback(cfg *config.Config, c *gin.Context) bool {
	if cfg == nil {
		return false
	}
	secret := strings.TrimSpace(cfg.VideoCallbackSecret)
	if secret == "" {
		// Fail closed in production; allow local/dev without extra setup.
		return cfg.Environment != "production"
	}
	// Header only — never accept ?secret= (lands in access logs / referrers).
	provided := strings.TrimSpace(firstNonEmpty(
		c.GetHeader("X-Callback-Secret"),
		c.GetHeader("X-Video-Callback-Secret"),
	))
	if provided == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(secret)) == 1
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	default:
		return ""
	}
}

// --- Direct Link Proxy ---

