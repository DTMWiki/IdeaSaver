package repository

import (
	"context"
	"database/sql"

	"github.com/DTMWiki/IdeaSaver/server/internal/model"
	"github.com/google/uuid"
)

// ShareRepository handles share link database operations.
type ShareRepository struct {
	db *sql.DB
}

func NewShareRepository(db *sql.DB) *ShareRepository {
	return &ShareRepository{db: db}
}

func (r *ShareRepository) Create(ctx context.Context, s *model.Share) error {
	return r.db.QueryRowContext(ctx,
		`INSERT INTO shares (user_id, file_id, code, password, expires_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at`,
		s.UserID, s.FileID, s.Code, s.Password, s.ExpiresAt,
	).Scan(&s.ID, &s.CreatedAt)
}

func (r *ShareRepository) FindByCode(ctx context.Context, code string) (*model.Share, error) {
	var s model.Share
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, file_id, code, password, expires_at, view_count, created_at
		 FROM shares WHERE code = $1`, code).Scan(
		&s.ID, &s.UserID, &s.FileID, &s.Code, &s.Password, &s.ExpiresAt,
		&s.ViewCount, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ShareRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Share, error) {
	var s model.Share
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, file_id, code, password, expires_at, view_count, created_at
		 FROM shares WHERE id = $1`, id).Scan(
		&s.ID, &s.UserID, &s.FileID, &s.Code, &s.Password, &s.ExpiresAt,
		&s.ViewCount, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ShareRepository) IncrementViewCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE shares SET view_count = view_count + 1 WHERE id = $1`, id)
	return err
}

func (r *ShareRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.Share, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT s.id, s.user_id, s.file_id, s.code, s.password, s.expires_at, s.view_count, s.created_at,
		        COALESCE(f.name, '') AS file_name,
		        CASE WHEN COALESCE(s.password, '') <> '' THEN TRUE ELSE FALSE END AS has_password
		 FROM shares s
		 LEFT JOIN files f ON f.id = s.file_id
		 WHERE s.user_id = $1
		 ORDER BY s.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []model.Share
	for rows.Next() {
		var s model.Share
		if err := rows.Scan(&s.ID, &s.UserID, &s.FileID, &s.Code, &s.Password,
			&s.ExpiresAt, &s.ViewCount, &s.CreatedAt, &s.FileName, &s.HasPassword); err != nil {
			return nil, err
		}
		// Never expose raw password material in list payloads.
		s.Password = ""
		shares = append(shares, s)
	}
	return shares, nil
}

func (r *ShareRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM shares WHERE id = $1`, id)
	return err
}
