package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/DTMWiki/IdeaSaver/server/internal/model"
	"github.com/google/uuid"
)

// VideoRepository handles video database operations.
type VideoRepository struct {
	db *sql.DB
}

func NewVideoRepository(db *sql.DB) *VideoRepository {
	return &VideoRepository{db: db}
}

func (r *VideoRepository) Create(ctx context.Context, v *model.Video) error {
	return r.db.QueryRowContext(ctx,
		`INSERT INTO videos (user_id, title, vid, vcode, player_user_id, thumbnail_url, thumbnail_small_url, play_count, transcode_status, transcode_message, status, play_url, size)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		 RETURNING id, created_at, updated_at`,
		v.UserID, v.Title, v.VID, v.VCode, v.PlayerUserID, v.ThumbnailURL, v.ThumbnailSmallURL, v.PlayCount, v.TranscodeStatus, v.TranscodeMessage, v.Status, v.PlayURL, v.Size,
	).Scan(&v.ID, &v.CreatedAt, &v.UpdatedAt)
}

func (r *VideoRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Video, error) {
	var v model.Video
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, title, vid, vcode, player_user_id, thumbnail_url, thumbnail_small_url, play_count, transcode_status, transcode_message, status, play_url, size, created_at, updated_at
		 FROM videos WHERE id = $1`, id).Scan(
		&v.ID, &v.UserID, &v.Title, &v.VID, &v.VCode, &v.PlayerUserID, &v.ThumbnailURL, &v.ThumbnailSmallURL, &v.PlayCount, &v.TranscodeStatus, &v.TranscodeMessage, &v.Status, &v.PlayURL, &v.Size,
		&v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *VideoRepository) ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]model.Video, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM videos WHERE user_id = $1`, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, title, vid, vcode, player_user_id, thumbnail_url, thumbnail_small_url, play_count, transcode_status, transcode_message, status, play_url, size, created_at, updated_at
		 FROM videos WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var videos []model.Video
	for rows.Next() {
		var v model.Video
		if err := rows.Scan(&v.ID, &v.UserID, &v.Title, &v.VID, &v.VCode, &v.PlayerUserID, &v.ThumbnailURL, &v.ThumbnailSmallURL, &v.PlayCount,
			&v.TranscodeStatus, &v.TranscodeMessage, &v.Status, &v.PlayURL, &v.Size, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, 0, err
		}
		videos = append(videos, v)
	}
	return videos, total, nil
}

func (r *VideoRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status int16) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE videos SET status = $2, updated_at = NOW() WHERE id = $1`, id, status)
	return err
}

func (r *VideoRepository) UpdatePlayURL(ctx context.Context, vid string, playURL string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE videos SET play_url = $2, updated_at = NOW() WHERE vid = $1`, vid, playURL)
	return err
}

func (r *VideoRepository) UpdatePlaybackMeta(ctx context.Context, vid, vcode, playerUserID, playURL, thumbnailURL, thumbnailSmallURL string, playCount int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE videos
		 SET vcode = CASE WHEN $2 <> '' THEN $2 ELSE vcode END,
		     player_user_id = CASE WHEN $3 <> '' THEN $3 ELSE player_user_id END,
		     play_url = CASE WHEN $4 <> '' THEN $4 ELSE play_url END,
		     thumbnail_url = CASE WHEN $5 <> '' THEN $5 ELSE thumbnail_url END,
		     thumbnail_small_url = CASE WHEN $6 <> '' THEN $6 ELSE thumbnail_small_url END,
		     play_count = CASE WHEN $7 >= 0 THEN $7 ELSE play_count END,
		     updated_at = NOW()
		 WHERE vid = $1`,
		vid, vcode, playerUserID, playURL, thumbnailURL, thumbnailSmallURL, playCount,
	)
	return err
}

func (r *VideoRepository) UpdateTranscodeState(ctx context.Context, vid, transcodeStatus, transcodeMessage string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE videos
		 SET transcode_status = CASE WHEN $2 <> '' THEN $2 ELSE transcode_status END,
		     transcode_message = CASE WHEN $3 <> '' THEN $3 ELSE transcode_message END,
		     updated_at = NOW()
		 WHERE vid = $1`,
		vid, transcodeStatus, transcodeMessage,
	)
	return err
}

func (r *VideoRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM videos WHERE id = $1`, id)
	return err
}

func (r *VideoRepository) BatchDelete(ctx context.Context, ids []uuid.UUID) error {
	for _, id := range ids {
		if _, err := r.db.ExecContext(ctx, `DELETE FROM videos WHERE id = $1`, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *VideoRepository) FindByVID(ctx context.Context, vid string) (*model.Video, error) {
	var v model.Video
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, title, vid, vcode, player_user_id, thumbnail_url, thumbnail_small_url, play_count, transcode_status, transcode_message, status, play_url, size, created_at, updated_at
		 FROM videos WHERE vid = $1`, vid).Scan(
		&v.ID, &v.UserID, &v.Title, &v.VID, &v.VCode, &v.PlayerUserID, &v.ThumbnailURL, &v.ThumbnailSmallURL, &v.PlayCount, &v.TranscodeStatus, &v.TranscodeMessage, &v.Status, &v.PlayURL, &v.Size,
		&v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ListAll returns all videos across all users (for admin).
func (r *VideoRepository) ListAll(ctx context.Context, keyword string, offset, limit int) ([]model.Video, int, error) {
	keyword = strings.TrimSpace(keyword)
	countQuery := `SELECT COUNT(*)
		FROM videos v
		LEFT JOIN users u ON u.id = v.user_id`
	args := []any{}
	clauses := []string{}
	if keyword != "" {
		args = append(args, "%"+keyword+"%")
		clauses = append(clauses, `(v.title ILIKE $1 OR COALESCE(v.vcode, '') ILIKE $1 OR COALESCE(v.vid, '') ILIKE $1 OR COALESCE(u.username, '') ILIKE $1)`)
	}
	if len(clauses) > 0 {
		countQuery += " WHERE " + strings.Join(clauses, " AND ")
	}

	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	listQuery := `SELECT v.id, v.user_id, v.title, v.vid, v.vcode, v.player_user_id, v.thumbnail_url, v.thumbnail_small_url, v.play_count,
			v.transcode_status, v.transcode_message, v.status, v.play_url, v.size, v.created_at, v.updated_at, COALESCE(u.username, '')
		 FROM videos v
		 LEFT JOIN users u ON u.id = v.user_id`
	if len(clauses) > 0 {
		listQuery += " WHERE " + strings.Join(clauses, " AND ")
	}
	listQuery += fmt.Sprintf(" ORDER BY v.created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var videos []model.Video
	for rows.Next() {
		var v model.Video
		if err := rows.Scan(&v.ID, &v.UserID, &v.Title, &v.VID, &v.VCode, &v.PlayerUserID, &v.ThumbnailURL, &v.ThumbnailSmallURL, &v.PlayCount,
			&v.TranscodeStatus, &v.TranscodeMessage, &v.Status, &v.PlayURL, &v.Size, &v.CreatedAt, &v.UpdatedAt, &v.Username); err != nil {
			return nil, 0, err
		}
		videos = append(videos, v)
	}
	return videos, total, nil
}
