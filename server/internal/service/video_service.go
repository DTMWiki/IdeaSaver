package service

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/DTMWiki/IdeaSaver/server/internal/model"
	"github.com/DTMWiki/IdeaSaver/server/internal/repository"
	"github.com/DTMWiki/IdeaSaver/server/internal/storage"
	"github.com/google/uuid"
)

const (
	videoTranscodePending    = "pending"
	videoTranscodeProcessing = "processing"
	videoTranscodeReady      = "ready"
	videoTranscodeFailed     = "failed"
	videoTranscodeBlocked    = "blocked"
)

// VideoService handles video management business logic.
type VideoService struct {
	cfg    *config.Config
	repos  *repository.Repositories
	vcloud *storage.VCloudClient
	sse    *SSEService
}

type VideoCallbackPayload struct {
	Msg          string
	VID          string
	VCode        string
	PlayerUserID string
	Callback     string // callbackString
}

type VideoPlayInfo struct {
	Ready           bool   `json:"ready"`
	PlayURL         string `json:"play_url,omitempty"`
	VCode           string `json:"vcode,omitempty"`
	PlayerUserID    string `json:"player_user_id,omitempty"`
	TranscodeStatus string `json:"transcode_status"`
	Message         string `json:"message,omitempty"`
}

func NewVideoService(cfg *config.Config, repos *repository.Repositories, vcloud *storage.VCloudClient, sse *SSEService) *VideoService {
	return &VideoService{cfg: cfg, repos: repos, vcloud: vcloud, sse: sse}
}

// UploadVideo uploads a video to DogeCloud VCloud.
func (s *VideoService) UploadVideo(ctx context.Context, userID uuid.UUID, title string, reader io.Reader, filename string, size int64) (*model.Video, error) {
	callbackStr := fmt.Sprintf("user:%s", userID.String())

	vid, err := s.vcloud.UploadVideo(title, reader, filename, size, callbackStr)
	if err != nil {
		return nil, fmt.Errorf("failed to upload video: %w", err)
	}

	video := &model.Video{
		UserID:           userID,
		Title:            title,
		VID:              vid,
		TranscodeStatus:  videoTranscodeProcessing,
		TranscodeMessage: "视频上传完成，等待转码",
		Status:           1, // enabled by default
		Size:             size,
	}

	video, _ = s.refreshPlaybackMeta(ctx, video, "", "")

	if err := s.repos.Videos.Create(ctx, video); err != nil {
		return nil, err
	}

	_ = s.repos.AuditLogs.Create(ctx, &model.AuditLog{
		UserID:     userID,
		Action:     "video_upload",
		Resource:   "video",
		ResourceID: &video.ID,
		Details: map[string]any{
			"title":  video.Title,
			"vid":    video.VID,
			"vcode":  video.VCode,
			"status": video.TranscodeStatus,
			"size":   video.Size,
		},
	})

	return video, nil
}

// HandleCallback processes DogeCloud upload/transcode callback.
func (s *VideoService) HandleCallback(ctx context.Context, payload VideoCallbackPayload) error {
	vid := strings.TrimSpace(payload.VID)
	if vid == "" {
		return fmt.Errorf("missing vid in callback")
	}

	video, err := s.repos.Videos.FindByVID(ctx, vid)
	if err != nil {
		return fmt.Errorf("video not found: %w", err)
	}

	msg := strings.TrimSpace(payload.Msg)

	video.VCode = firstNonEmptyTrim(payload.VCode, video.VCode)
	video.PlayerUserID = firstNonEmptyTrim(s.cfg.DogeUserID, payload.PlayerUserID, video.PlayerUserID)
	video.TranscodeStatus, video.TranscodeMessage = callbackState(msg, video.TranscodeStatus)

	_ = s.repos.Videos.UpdatePlaybackMeta(ctx, video.VID, video.VCode, video.PlayerUserID, "", "", "")
	_ = s.repos.Videos.UpdateTranscodeState(ctx, video.VID, video.TranscodeStatus, video.TranscodeMessage)

	video, _ = s.refreshPlaybackMeta(ctx, video, "", "")
	_ = s.repos.Videos.UpdateTranscodeState(ctx, video.VID, video.TranscodeStatus, video.TranscodeMessage)

	eventType := "video_status_update"
	switch msg {
	case "upload":
		eventType = "video_upload_complete"
	case "transcode":
		eventType = "video_transcode_complete"
	case "transcode_failed":
		eventType = "video_transcode_failed"
	case "blocked":
		eventType = "video_blocked"
	}

	s.sse.SendToUser(video.UserID, SSEEvent{
		Type: eventType,
		Data: map[string]any{
			"video_id":          video.ID,
			"title":             video.Title,
			"play_url":          video.PlayURL,
			"vcode":             video.VCode,
			"player_user_id":    video.PlayerUserID,
			"thumbnail_url":     video.ThumbnailURL,
			"thumbnail_small":   video.ThumbnailSmallURL,
			"transcode_status":  video.TranscodeStatus,
			"transcode_message": video.TranscodeMessage,
			"callback_msg":      msg,
			"callback_string":   strings.TrimSpace(payload.Callback),
		},
	})
	if msg == "transcode" {
		s.sse.SendToUser(video.UserID, SSEEvent{
			Type: "video_ready",
			Data: map[string]any{
				"video_id":          video.ID,
				"title":             video.Title,
				"play_url":          video.PlayURL,
				"vcode":             video.VCode,
				"player_user_id":    video.PlayerUserID,
				"thumbnail_url":     video.ThumbnailURL,
				"thumbnail_small":   video.ThumbnailSmallURL,
				"transcode_status":  video.TranscodeStatus,
				"transcode_message": video.TranscodeMessage,
			},
		})
	}

	return nil
}

// ListVideos returns the user's videos.
func (s *VideoService) ListVideos(ctx context.Context, userID uuid.UUID, offset, limit int) ([]model.Video, int, error) {
	videos, total, err := s.repos.Videos.ListByUser(ctx, userID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	refreshed := 0
	for i := range videos {
		if !shouldRefreshVideoMeta(&videos[i]) {
			continue
		}
		if refreshed >= 6 {
			break
		}
		if updated, err := s.refreshPlaybackMeta(ctx, &videos[i], "", ""); err == nil && updated != nil {
			videos[i] = *updated
		}
		refreshed++
	}

	return videos, total, nil
}

// SetVideoStatus enables or disables a video.
func (s *VideoService) SetVideoStatus(ctx context.Context, id uuid.UUID, userID uuid.UUID, status int16) error {
	video, err := s.repos.Videos.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if video.UserID != userID {
		return fmt.Errorf("permission denied")
	}

	dogeStatus := 0
	if status == 1 {
		dogeStatus = 1
	}
	if err := s.vcloud.SetVideoStatus([]string{video.VID}, dogeStatus); err != nil {
		return fmt.Errorf("failed to update status on DogeCloud: %w", err)
	}

	if err := s.repos.Videos.UpdateStatus(ctx, id, status); err != nil {
		return err
	}

	_ = s.repos.AuditLogs.Create(ctx, &model.AuditLog{
		UserID:     userID,
		Action:     "video_status_change",
		Resource:   "video",
		ResourceID: &video.ID,
		Details: map[string]any{
			"title":  video.Title,
			"vid":    video.VID,
			"status": status,
		},
	})

	return nil
}

// DeleteVideo deletes a single video.
func (s *VideoService) DeleteVideo(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	video, err := s.repos.Videos.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if video.UserID != userID {
		return fmt.Errorf("permission denied")
	}

	_ = s.vcloud.DeleteVideos([]string{video.VID})

	if err := s.repos.Videos.Delete(ctx, id); err != nil {
		return err
	}

	_ = s.repos.AuditLogs.Create(ctx, &model.AuditLog{
		UserID:     userID,
		Action:     "video_delete",
		Resource:   "video",
		ResourceID: &video.ID,
		Details: map[string]any{
			"title": video.Title,
			"vid":   video.VID,
		},
	})

	return nil
}

// BatchDeleteVideos deletes multiple videos.
func (s *VideoService) BatchDeleteVideos(ctx context.Context, ids []uuid.UUID, userID uuid.UUID) error {
	var vids []string
	var deleted []map[string]any
	for _, id := range ids {
		video, err := s.repos.Videos.FindByID(ctx, id)
		if err != nil {
			continue
		}
		if video.UserID != userID {
			return fmt.Errorf("permission denied for video %s", id)
		}
		vids = append(vids, video.VID)
		deleted = append(deleted, map[string]any{
			"id":    video.ID.String(),
			"title": video.Title,
			"vid":   video.VID,
		})
	}

	if len(vids) > 0 {
		_ = s.vcloud.DeleteVideos(vids)
	}

	if err := s.repos.Videos.BatchDelete(ctx, ids); err != nil {
		return err
	}

	_ = s.repos.AuditLogs.Create(ctx, &model.AuditLog{
		UserID:   userID,
		Action:   "video_delete",
		Resource: "video",
		Details: map[string]any{
			"count":  len(deleted),
			"videos": deleted,
		},
	})

	return nil
}

func (s *VideoService) GetPlayInfo(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*VideoPlayInfo, error) {
	return s.GetPlayInfoForViewer(ctx, id, userID, "", "")
}

func (s *VideoService) GetPlayInfoForViewer(ctx context.Context, id uuid.UUID, userID uuid.UUID, viewerIP, userAgent string) (*VideoPlayInfo, error) {
	video, err := s.repos.Videos.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if video.UserID != userID {
		return nil, fmt.Errorf("permission denied")
	}

	video, _ = s.refreshPlaybackMeta(ctx, video, viewerIP, userAgent)

	info := &VideoPlayInfo{
		PlayURL:         strings.TrimSpace(video.PlayURL),
		VCode:           strings.TrimSpace(video.VCode),
		PlayerUserID:    strings.TrimSpace(video.PlayerUserID),
		TranscodeStatus: normalizeTranscodeStatus(video.TranscodeStatus),
	}
	info.Ready = info.TranscodeStatus == videoTranscodeReady && canPlayVideo(video)
	if !info.Ready {
		info.Message = playbackMessage(video)
	}
	return info, nil
}

// GetPlayURL returns a playable URL for backward compatibility.
func (s *VideoService) GetPlayURL(ctx context.Context, id uuid.UUID, userID uuid.UUID) (string, error) {
	info, err := s.GetPlayInfo(ctx, id, userID)
	if err != nil {
		return "", err
	}
	if !info.Ready {
		return "", fmt.Errorf(firstNonEmptyTrim(info.Message, "播放地址尚未就绪"))
	}
	if strings.TrimSpace(info.PlayURL) == "" {
		return "", fmt.Errorf("播放地址尚未就绪")
	}
	return info.PlayURL, nil
}

func (s *VideoService) refreshPlaybackMeta(ctx context.Context, video *model.Video, viewerIP, userAgent string) (*model.Video, error) {
	if video == nil {
		return nil, fmt.Errorf("nil video")
	}

	if info, err := s.vcloud.GetVideoInfo(video.VID); err == nil && info != nil {
		video.VCode = firstNonEmptyTrim(info.VCode, video.VCode)
		video.PlayerUserID = firstNonEmptyTrim(s.cfg.DogeUserID, info.PlayerUserID, video.PlayerUserID)
		video.PlayURL = firstNonEmptyTrim(info.PlayURL, video.PlayURL)
		video.ThumbnailURL = firstNonEmptyTrim(info.ThumbnailURL, video.ThumbnailURL)
		video.ThumbnailSmallURL = firstNonEmptyTrim(info.ThumbnailSmallURL, video.ThumbnailSmallURL)
		video.TranscodeStatus = mergeVideoStatus(video.TranscodeStatus, info.Status)
	}

	if strings.TrimSpace(viewerIP) != "" && strings.TrimSpace(video.VCode) != "" {
		if playURL, err := s.vcloud.GetBestPlayURL(video.VCode, viewerIP, userAgent); err == nil {
			video.PlayURL = firstNonEmptyTrim(playURL, video.PlayURL)
		}
	}
	video.PlayerUserID = firstNonEmptyTrim(s.cfg.DogeUserID, video.PlayerUserID, playerUserIDFromURL(video.PlayURL))

	if strings.TrimSpace(video.PlayURL) == "" && strings.TrimSpace(video.VCode) != "" && strings.TrimSpace(video.PlayerUserID) != "" {
		video.PlayURL = firstNonEmptyTrim(video.PlayURL, s.vcloud.BuildPlayerMP4URL(video.VCode, video.PlayerUserID))
	}

	if err := s.repos.Videos.UpdatePlaybackMeta(ctx, video.VID, video.VCode, video.PlayerUserID, video.PlayURL, video.ThumbnailURL, video.ThumbnailSmallURL); err != nil {
		return nil, err
	}
	if err := s.repos.Videos.UpdateTranscodeState(ctx, video.VID, normalizeTranscodeStatus(video.TranscodeStatus), strings.TrimSpace(video.TranscodeMessage)); err != nil {
		return nil, err
	}

	return video, nil
}

func shouldRefreshVideoMeta(video *model.Video) bool {
	if video == nil {
		return false
	}
	if strings.TrimSpace(video.ThumbnailSmallURL) == "" || strings.TrimSpace(video.ThumbnailURL) == "" {
		return true
	}
	if strings.TrimSpace(video.VCode) == "" || strings.TrimSpace(video.PlayerUserID) == "" {
		return true
	}
	status := normalizeTranscodeStatus(video.TranscodeStatus)
	return status == videoTranscodePending || status == videoTranscodeProcessing
}

func canPlayVideo(video *model.Video) bool {
	if video == nil {
		return false
	}
	return strings.TrimSpace(video.VCode) != "" && strings.TrimSpace(video.PlayerUserID) != ""
}

func callbackState(msg, current string) (string, string) {
	switch strings.TrimSpace(msg) {
	case "upload":
		return videoTranscodeProcessing, "视频上传完成，等待转码"
	case "transcode":
		return videoTranscodeReady, "视频转码完成，可开始播放"
	case "transcode_failed":
		return videoTranscodeFailed, "视频转码失败"
	case "blocked":
		return videoTranscodeBlocked, "视频因审核或策略原因暂不可播放"
	default:
		if normalized := normalizeTranscodeStatus(current); normalized != "" {
			return normalized, playbackMessage(&model.Video{TranscodeStatus: normalized})
		}
		return videoTranscodeProcessing, "视频状态同步中"
	}
}

func normalizeTranscodeStatus(status string) string {
	switch strings.TrimSpace(status) {
	case videoTranscodeReady, videoTranscodeFailed, videoTranscodeBlocked, videoTranscodeProcessing:
		return strings.TrimSpace(status)
	case "", videoTranscodePending:
		return videoTranscodePending
	default:
		return videoTranscodePending
	}
}

func mergeVideoStatus(current string, dogeStatus int) string {
	switch dogeStatus {
	case 20:
		return videoTranscodeFailed
	case 21, 40:
		return videoTranscodeBlocked
	case 30, 31:
		if normalizeTranscodeStatus(current) == videoTranscodeReady {
			return videoTranscodeReady
		}
		return videoTranscodeProcessing
	case 10:
		return normalizeTranscodeStatus(current)
	default:
		return normalizeTranscodeStatus(current)
	}
}

func playbackMessage(video *model.Video) string {
	if video == nil {
		return "视频状态未知，请稍后重试"
	}

	switch normalizeTranscodeStatus(video.TranscodeStatus) {
	case videoTranscodeReady:
		if !canPlayVideo(video) {
			return "已收到转码成功回调，正在同步播放信息"
		}
		return firstNonEmptyTrim(video.TranscodeMessage, "视频已就绪")
	case videoTranscodeFailed:
		return firstNonEmptyTrim(video.TranscodeMessage, "视频转码失败，请重新上传或联系管理员")
	case videoTranscodeBlocked:
		return firstNonEmptyTrim(video.TranscodeMessage, "视频已被屏蔽，暂不可播放")
	case videoTranscodeProcessing, videoTranscodePending:
		return firstNonEmptyTrim(video.TranscodeMessage, "视频转码中，请稍后重试")
	default:
		return "视频状态同步中，请稍后重试"
	}
}

func firstNonEmptyTrim(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func playerUserIDFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	q := u.Query()
	return firstNonEmptyTrim(q.Get("userId"), q.Get("userid"), q.Get("uid"))
}
