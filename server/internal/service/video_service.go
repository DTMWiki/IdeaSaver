package service

import (
	"context"
	"fmt"
	"io"

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

	if err := s.repos.Videos.Create(ctx, video); err != nil {
		return nil, err
	}

	return video, nil
}

// HandleCallback processes DogeCloud upload/transcode callback.
func (s *VideoService) HandleCallback(ctx context.Context, msg, vid, callback string) error {
	video, err := s.repos.Videos.FindByVID(ctx, vid)
	if err != nil {
		return fmt.Errorf("video not found: %w", err)
	}

	if msg == "upload" || msg == "transcode" {
		// Get playback URL
		streams, err := s.vcloud.GetVideoStreams(video.VCode)
		if err == nil {
			if data, ok := streams["data"].(map[string]any); ok {
				if playURL, ok := data["play_url"].(string); ok {
					_ = s.repos.Videos.UpdatePlayURL(ctx, vid, playURL)

					// Notify user via SSE
					s.sse.SendToUser(video.UserID, SSEEvent{
						Type: "video_ready",
						Data: map[string]any{
							"video_id": video.ID,
							"title":    video.Title,
							"play_url": playURL,
						},
					})
				}
			}
		}
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

// GetPlayURL returns the playback URL for a video.
func (s *VideoService) GetPlayURL(ctx context.Context, id uuid.UUID, userID uuid.UUID) (string, error) {
	video, err := s.repos.Videos.FindByID(ctx, id)
	if err != nil {
		return "", err
	}
	if video.UserID != userID {
		return "", fmt.Errorf("permission denied")
	}

	if video.PlayURL != "" {
		return video.PlayURL, nil
	}

	// Try to fetch from DogeCloud
	if video.VCode != "" {
		streams, err := s.vcloud.GetVideoStreams(video.VCode)
		if err == nil {
			if data, ok := streams["data"].(map[string]any); ok {
				if playURL, ok := data["play_url"].(string); ok {
					_ = s.repos.Videos.UpdatePlayURL(ctx, video.VID, playURL)
					return playURL, nil
				}
			}
		}
	}

	return "", fmt.Errorf("播放地址尚未就绪")
}
