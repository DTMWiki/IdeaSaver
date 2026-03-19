package repository

import (
	"context"
	"database/sql"

	"github.com/DTMWiki/IdeaSaver/server/internal/model"
	"github.com/google/uuid"
)

// FileAppealRepository handles moderation appeal ticket operations.
type FileAppealRepository struct {
	db *sql.DB
}

func NewFileAppealRepository(db *sql.DB) *FileAppealRepository {
	return &FileAppealRepository{db: db}
}

func (r *FileAppealRepository) Create(ctx context.Context, appeal *model.FileAppeal) error {
	status := appeal.Status
	if status == "" {
		status = "pending"
	}

	return r.db.QueryRowContext(ctx,
		`INSERT INTO file_appeals (file_id, user_id, status, reason)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at, updated_at`,
		appeal.FileID, appeal.UserID, status, appeal.Reason,
	).Scan(&appeal.ID, &appeal.CreatedAt, &appeal.UpdatedAt)
}

func (r *FileAppealRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.FileAppeal, error) {
	var appeal model.FileAppeal
	err := r.db.QueryRowContext(ctx,
		`SELECT id, file_id, user_id, status, reason, admin_comment, reviewed_by, reviewed_at, created_at, updated_at
		 FROM file_appeals WHERE id = $1`, id).Scan(
		&appeal.ID, &appeal.FileID, &appeal.UserID, &appeal.Status, &appeal.Reason,
		&appeal.AdminComment, &appeal.ReviewedBy, &appeal.ReviewedAt, &appeal.CreatedAt, &appeal.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &appeal, nil
}

func (r *FileAppealRepository) FindPendingByFileID(ctx context.Context, fileID uuid.UUID) (*model.FileAppeal, error) {
	var appeal model.FileAppeal
	err := r.db.QueryRowContext(ctx,
		`SELECT id, file_id, user_id, status, reason, admin_comment, reviewed_by, reviewed_at, created_at, updated_at
		 FROM file_appeals
		 WHERE file_id = $1 AND status = 'pending'
		 ORDER BY created_at DESC
		 LIMIT 1`, fileID).Scan(
		&appeal.ID, &appeal.FileID, &appeal.UserID, &appeal.Status, &appeal.Reason,
		&appeal.AdminComment, &appeal.ReviewedBy, &appeal.ReviewedAt, &appeal.CreatedAt, &appeal.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &appeal, nil
}

func (r *FileAppealRepository) List(ctx context.Context, status string, offset, limit int) ([]model.FileAppeal, int, error) {
	var total int

	if status != "" {
		if err := r.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM file_appeals WHERE status = $1`, status,
		).Scan(&total); err != nil {
			return nil, 0, err
		}
	} else {
		if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM file_appeals`).Scan(&total); err != nil {
			return nil, 0, err
		}
	}

	var (
		rows *sql.Rows
		err  error
	)

	baseQuery := `SELECT fa.id, fa.file_id, fa.user_id, fa.status, fa.reason, fa.admin_comment,
		                 fa.reviewed_by, fa.reviewed_at, fa.created_at, fa.updated_at,
		                 f.name AS file_name, u.username, COALESCE(ru.username, '')
		          FROM file_appeals fa
		          JOIN files f ON fa.file_id = f.id
		          JOIN users u ON fa.user_id = u.id
		          LEFT JOIN users ru ON fa.reviewed_by = ru.id`

	if status != "" {
		rows, err = r.db.QueryContext(ctx,
			baseQuery+`
		          WHERE fa.status = $1
		          ORDER BY fa.created_at DESC
		          LIMIT $2 OFFSET $3`,
			status, limit, offset,
		)
	} else {
		rows, err = r.db.QueryContext(ctx,
			baseQuery+`
		          ORDER BY fa.created_at DESC
		          LIMIT $1 OFFSET $2`,
			limit, offset,
		)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	appeals := make([]model.FileAppeal, 0, limit)
	for rows.Next() {
		var appeal model.FileAppeal
		if err := rows.Scan(
			&appeal.ID, &appeal.FileID, &appeal.UserID, &appeal.Status, &appeal.Reason, &appeal.AdminComment,
			&appeal.ReviewedBy, &appeal.ReviewedAt, &appeal.CreatedAt, &appeal.UpdatedAt,
			&appeal.FileName, &appeal.Username, &appeal.ReviewerName,
		); err != nil {
			return nil, 0, err
		}
		appeals = append(appeals, appeal)
	}
	return appeals, total, nil
}

func (r *FileAppealRepository) Review(ctx context.Context, id uuid.UUID, status, adminComment string, reviewedBy uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE file_appeals
		 SET status = $2,
		     admin_comment = $3,
		     reviewed_by = $4,
		     reviewed_at = NOW(),
		     updated_at = NOW()
		 WHERE id = $1`,
		id, status, adminComment, reviewedBy,
	)
	return err
}
