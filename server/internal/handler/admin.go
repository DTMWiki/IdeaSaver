package handler

import (
	"net/http"
	"strings"

	"github.com/DTMWiki/IdeaSaver/server/internal/middleware"
	"github.com/DTMWiki/IdeaSaver/server/internal/repository"
	"github.com/DTMWiki/IdeaSaver/server/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func handleAdminListFiles(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		offset, limit := parseOffsetLimit(c, 50, 200)
		keyword := strings.TrimSpace(c.Query("keyword"))

		files, total, err := svc.Admin.ListAllFiles(c.Request.Context(), keyword, offset, limit)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"files": files, "total": total})
	}
}

func handleAdminDeleteFile(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件ID"})
			return
		}
		if err := svc.Admin.DeleteFile(c.Request.Context(), id, admin.ID); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleAdminBanFile(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件ID"})
			return
		}

		var req struct {
			Reason string `json:"reason"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
			return
		}

		if err := svc.Admin.BanFile(c.Request.Context(), id, admin.ID, req.Reason); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleAdminUnbanFile(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件ID"})
			return
		}

		var req struct {
			Comment string `json:"comment"`
		}
		_ = c.ShouldBindJSON(&req)

		if err := svc.Admin.UnbanFile(c.Request.Context(), id, admin.ID, req.Comment); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleAdminListVideos(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		offset, limit := parseOffsetLimit(c, 50, 200)
		keyword := strings.TrimSpace(c.Query("keyword"))

		videos, total, err := svc.Admin.ListAllVideos(c.Request.Context(), keyword, offset, limit)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"videos": videos, "total": total})
	}
}

func handleAdminDeleteVideo(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的视频ID"})
			return
		}
		if err := svc.Admin.DeleteVideo(c.Request.Context(), id, admin.ID); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleAdminGetVideoPlayInfo(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的视频ID"})
			return
		}

		info, err := svc.Video.GetPlayInfoForActor(c.Request.Context(), id, admin, c.ClientIP(), c.Request.UserAgent())
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, info)
	}
}

func handleAdminSetVideoStatus(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := middleware.GetUser(c)
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

		if err := svc.Video.SetVideoStatusForActor(c.Request.Context(), id, admin, req.Status); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleAdminListAppeals(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.Query("status")
		offset, limit := parseOffsetLimit(c, 50, 200)

		appeals, total, err := svc.Admin.ListAppeals(c.Request.Context(), status, offset, limit)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"appeals": appeals, "total": total})
	}
}

func handleAdminReviewAppeal(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := middleware.GetUser(c)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的工单ID"})
			return
		}

		var req struct {
			Decision string `json:"decision"`
			Comment  string `json:"comment"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
			return
		}

		if err := svc.Admin.ReviewAppeal(c.Request.Context(), id, admin.ID, req.Decision, req.Comment); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleAdminListLogs(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		offset, limit := parseOffsetLimit(c, 50, 200)
		startAt, err := parseAuditLogTime(c.Query("start_at"), false)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始时间"})
			return
		}
		endAt, err := parseAuditLogTime(c.Query("end_at"), true)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束时间"})
			return
		}

		filter := repository.AuditLogFilter{
			Action:  c.Query("action"),
			User:    c.Query("user"),
			Keyword: c.Query("keyword"),
			StartAt: startAt,
			EndAt:   endAt,
		}

		logs, total, err := svc.Admin.ListAuditLogs(c.Request.Context(), filter, offset, limit)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"logs": logs, "total": total})
	}
}

func handleAdminListUsers(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		offset, limit := parseOffsetLimit(c, 50, 200)

		users, total, err := svc.Admin.ListUsers(c.Request.Context(), offset, limit)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"users": users, "total": total})
	}
}

func handleAdminUpdateQuota(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
			return
		}
		var req struct {
			Quota int64 `json:"quota"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
			return
		}

		if err := svc.Admin.UpdateUserQuota(c.Request.Context(), id, req.Quota); err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func handleAdminCleanupTrash(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		count, err := svc.Admin.CleanupTrash(c.Request.Context())
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"cleaned": count})
	}
}

func handleAdminRecalcStorage(svc *service.Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := middleware.GetUser(c)
		n, err := svc.Admin.RecalcAllStorageUsed(c.Request.Context(), admin.ID)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "users_updated": n})
	}
}

// --- Helpers ---

