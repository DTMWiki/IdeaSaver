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
	Ready        bool   `json:"ready"`
	PlayURL      string `json:"play_url,omitempty"`
	VCode        string `json:"vcode,omitempty"`
	PlayerUserID string `json:"player_user_id,omitempty"`
	Message      string `json:"message,omitempty"`
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
		UserID: userID,
		Title:  title,
		VID:    vid,
		Status: 1, // enabled by default
		Size:   size,
	}

	// Best-effort preload: fetch vcode/userId and possibly play URL immediately.
	if info, err := s.vcloud.GetVideoInfo(vid); err == nil && info != nil {
		video.VCode = firstNonEmptyTrim(info.VCode, video.VCode)
		video.PlayerUserID = firstNonEmptyTrim(info.PlayerUserID, video.PlayerUserID)
		video.PlayURL = firstNonEmptyTrim(info.PlayURL, video.PlayURL)
		video.ThumbnailURL = firstNonEmptyTrim(info.ThumbnailURL, video.ThumbnailURL)
		video.ThumbnailSmallURL = firstNonEmptyTrim(info.ThumbnailSmallURL, video.ThumbnailSmallURL)
	}
	video.PlayerUserID = firstNonEmptyTrim(video.PlayerUserID, playerUserIDFromURL(video.PlayURL), s.cfg.DogeUserID)
	if strings.TrimSpace(video.PlayURL) == "" {
		video.PlayURL = s.vcloud.BuildPlayerMP4URL(video.VCode, video.PlayerUserID)
	}

	if err := s.repos.Videos.Create(ctx, video); err != nil {
		return nil, err
	}

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

	// Callback may carry vcode / userId.
	video.VCode = firstNonEmptyTrim(payload.VCode, video.VCode)
	video.PlayerUserID = firstNonEmptyTrim(payload.PlayerUserID, video.PlayerUserID, s.cfg.DogeUserID)
	_ = s.repos.Videos.UpdatePlaybackMeta(ctx, video.VID, video.VCode, video.PlayerUserID, "", "", "")

	video, _ = s.refreshPlaybackMeta(ctx, video)

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
			"video_id":        video.ID,
			"title":           video.Title,
			"play_url":        video.PlayURL,
			"vcode":           video.VCode,
			"player_user_id":  video.PlayerUserID,
			"callback_msg":    msg,
			"callback_string": strings.TrimSpace(payload.Callback),
		},
	})
	if msg == "transcode" {
		// Backward compatibility for old listeners.
		s.sse.SendToUser(video.UserID, SSEEvent{
			Type: "video_ready",
			Data: map[string]any{
				"video_id":       video.ID,
				"title":          video.Title,
				"play_url":       video.PlayURL,
				"vcode":          video.VCode,
				"player_user_id": video.PlayerUserID,
			},
		})
	}

	return nil
}

// ListVideos returns the user's videos.
func (s *VideoService) ListVideos(ctx context.Context, userID uuid.UUID, offset, limit int) ([]model.Video, int, error) {
	return s.repos.Videos.ListByUser(ctx, userID, offset, limit)
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

	// Sync status to DogeCloud
	dogeStatus := 0
	if status == 1 {
		dogeStatus = 1
	}
	if err := s.vcloud.SetVideoStatus([]string{video.VID}, dogeStatus); err != nil {
		return fmt.Errorf("failed to update status on DogeCloud: %w", err)
	}

	return s.repos.Videos.UpdateStatus(ctx, id, status)
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

	// Delete from DogeCloud
	_ = s.vcloud.DeleteVideos([]string{video.VID})

	return s.repos.Videos.Delete(ctx, id)
}

// BatchDeleteVideos deletes multiple videos.
func (s *VideoService) BatchDeleteVideos(ctx context.Context, ids []uuid.UUID, userID uuid.UUID) error {
	var vids []string
	for _, id := range ids {
		video, err := s.repos.Videos.FindByID(ctx, id)
		if err != nil {
			continue
		}
		if video.UserID != userID {
			return fmt.Errorf("permission denied for video %s", id)
		}
		vids = append(vids, video.VID)
	}

	if len(vids) > 0 {
		_ = s.vcloud.DeleteVideos(vids)
	}

	return s.repos.Videos.BatchDelete(ctx, ids)
}

func (s *VideoService) GetPlayInfo(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*VideoPlayInfo, error) {
	video, err := s.repos.Videos.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if video.UserID != userID {
		return nil, fmt.Errorf("permission denied")
	}

	video, _ = s.refreshPlaybackMeta(ctx, video)

	info := &VideoPlayInfo{
		Ready:        strings.TrimSpace(video.PlayURL) != "",
		PlayURL:      strings.TrimSpace(video.PlayURL),
		VCode:        strings.TrimSpace(video.VCode),
		PlayerUserID: strings.TrimSpace(video.PlayerUserID),
	}
	if !info.Ready {
		info.Message = "视频仍在转码处理中，请稍后重试"
	}
	return info, nil
}

// GetPlayURL returns a playable URL for backward compatibility.
func (s *VideoService) GetPlayURL(ctx context.Context, id uuid.UUID, userID uuid.UUID) (string, error) {
	info, err := s.GetPlayInfo(ctx, id, userID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(info.PlayURL) == "" {
		return "", fmt.Errorf("播放地址尚未就绪")
	}
	return info.PlayURL, nil
}

func (s *VideoService) refreshPlaybackMeta(ctx context.Context, video *model.Video) (*model.Video, error) {
	if video == nil {
		return nil, fmt.Errorf("nil video")
	}

	// 1) Pull metadata by VID.
	if info, err := s.vcloud.GetVideoInfo(video.VID); err == nil && info != nil {
		video.VCode = firstNonEmptyTrim(info.VCode, video.VCode)
		video.PlayerUserID = firstNonEmptyTrim(info.PlayerUserID, video.PlayerUserID)
		video.PlayURL = firstNonEmptyTrim(info.PlayURL, video.PlayURL)
		video.ThumbnailURL = firstNonEmptyTrim(info.ThumbnailURL, video.ThumbnailURL)
		video.ThumbnailSmallURL = firstNonEmptyTrim(info.ThumbnailSmallURL, video.ThumbnailSmallURL)
	}

	// 2) Try streams API by vcode for direct play URL.
	if strings.TrimSpace(video.PlayURL) == "" && strings.TrimSpace(video.VCode) != "" {
		if playURL, err := s.vcloud.GetBestPlayURL(video.VCode); err == nil {
			video.PlayURL = firstNonEmptyTrim(playURL, video.PlayURL)
		}
	}
	video.PlayerUserID = firstNonEmptyTrim(video.PlayerUserID, playerUserIDFromURL(video.PlayURL), s.cfg.DogeUserID)

	// 3) Fallback player mp4 endpoint (vcode + userId).
	if strings.TrimSpace(video.PlayURL) == "" {
		video.PlayURL = firstNonEmptyTrim(video.PlayURL, s.vcloud.BuildPlayerMP4URL(video.VCode, video.PlayerUserID))
	}

	if err := s.repos.Videos.UpdatePlaybackMeta(ctx, video.VID, video.VCode, video.PlayerUserID, video.PlayURL, video.ThumbnailURL, video.ThumbnailSmallURL); err != nil {
		return nil, err
	}
	return video, nil
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
