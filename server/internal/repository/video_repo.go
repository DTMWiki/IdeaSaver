package repository

import (
	"context"
	"database/sql"

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
		`INSERT INTO videos (user_id, title, vid, vcode, status, play_url, size)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, created_at, updated_at`,
		v.UserID, v.Title, v.VID, v.VCode, v.Status, v.PlayURL, v.Size,
	).Scan(&v.ID, &v.CreatedAt, &v.UpdatedAt)
}

func (r *VideoRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Video, error) {
	var v model.Video
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, title, vid, vcode, status, play_url, size, created_at, updated_at
		 FROM videos WHERE id = $1`, id).Scan(
		&v.ID, &v.UserID, &v.Title, &v.VID, &v.VCode, &v.Status, &v.PlayURL, &v.Size,
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
		`SELECT id, user_id, title, vid, vcode, status, play_url, size, created_at, updated_at
		 FROM videos WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var videos []model.Video
	for rows.Next() {
		var v model.Video
		if err := rows.Scan(&v.ID, &v.UserID, &v.Title, &v.VID, &v.VCode,
			&v.Status, &v.PlayURL, &v.Size, &v.CreatedAt, &v.UpdatedAt); err != nil {
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
		`SELECT id, user_id, title, vid, vcode, status, play_url, size, created_at, updated_at
		 FROM videos WHERE vid = $1`, vid).Scan(
		&v.ID, &v.UserID, &v.Title, &v.VID, &v.VCode, &v.Status, &v.PlayURL, &v.Size,
		&v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ListAll returns all videos across all users (for admin).
func (r *VideoRepository) ListAll(ctx context.Context, offset, limit int) ([]model.Video, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM videos`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, title, vid, vcode, status, play_url, size, created_at, updated_at
		 FROM videos ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var videos []model.Video
	for rows.Next() {
		var v model.Video
		if err := rows.Scan(&v.ID, &v.UserID, &v.Title, &v.VID, &v.VCode,
			&v.Status, &v.PlayURL, &v.Size, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, 0, err
		}
		videos = append(videos, v)
	}
	return videos, total, nil
}
