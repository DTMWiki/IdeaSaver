package repository

import (
	"context"
	"database/sql"

	"github.com/DTMWiki/IdeaSaver/server/internal/model"
	"github.com/google/uuid"
)

// UploadTaskRepository handles upload task database operations.
type UploadTaskRepository struct {
	db *sql.DB
}

func NewUploadTaskRepository(db *sql.DB) *UploadTaskRepository {
	return &UploadTaskRepository{db: db}
}

func (r *UploadTaskRepository) Create(ctx context.Context, t *model.UploadTask) error {
	return r.db.QueryRowContext(ctx,
		`INSERT INTO upload_tasks (user_id, filename, total_size, chunk_size, total_chunks, storage_key, upload_id, status, target_type)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, created_at, updated_at`,
		t.UserID, t.Filename, t.TotalSize, t.ChunkSize, t.TotalChunks,
		t.StorageKey, t.UploadID, t.Status, t.TargetType,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *UploadTaskRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.UploadTask, error) {
	var t model.UploadTask
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, filename, total_size, uploaded_size, chunk_size, total_chunks,
		        uploaded_chunks, storage_key, upload_id, status, target_type, created_at, updated_at
		 FROM upload_tasks WHERE id = $1`, id).Scan(
		&t.ID, &t.UserID, &t.Filename, &t.TotalSize, &t.UploadedSize,
		&t.ChunkSize, &t.TotalChunks, &t.UploadedChunks, &t.StorageKey,
		&t.UploadID, &t.Status, &t.TargetType, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *UploadTaskRepository) UpdateChunkProgress(ctx context.Context, id uuid.UUID, uploadedChunks int, uploadedSize int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE upload_tasks SET uploaded_chunks = $2, uploaded_size = $3, status = 'uploading', updated_at = NOW()
		 WHERE id = $1`, id, uploadedChunks, uploadedSize)
	return err
}

func (r *UploadTaskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE upload_tasks SET status = $2, updated_at = NOW() WHERE id = $1`, id, status)
	return err
}

func (r *UploadTaskRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]model.UploadTask, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, filename, total_size, uploaded_size, chunk_size, total_chunks,
		        uploaded_chunks, storage_key, upload_id, status, target_type, created_at, updated_at
		 FROM upload_tasks WHERE user_id = $1 AND status NOT IN ('completed', 'failed')
		 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []model.UploadTask
	for rows.Next() {
		var t model.UploadTask
		if err := rows.Scan(&t.ID, &t.UserID, &t.Filename, &t.TotalSize, &t.UploadedSize,
			&t.ChunkSize, &t.TotalChunks, &t.UploadedChunks, &t.StorageKey,
			&t.UploadID, &t.Status, &t.TargetType, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (r *UploadTaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM upload_tasks WHERE id = $1`, id)
	return err
}
