package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"time"

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
	var partETagsRaw []byte
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, filename, total_size, uploaded_size, chunk_size, total_chunks,
		        uploaded_chunks, storage_key, upload_id, part_etags, status, target_type, created_at, updated_at
		 FROM upload_tasks WHERE id = $1`, id).Scan(
		&t.ID, &t.UserID, &t.Filename, &t.TotalSize, &t.UploadedSize,
		&t.ChunkSize, &t.TotalChunks, &t.UploadedChunks, &t.StorageKey,
		&t.UploadID, &partETagsRaw, &t.Status, &t.TargetType, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(partETagsRaw) > 0 {
		_ = json.Unmarshal(partETagsRaw, &t.PartETags)
	}
	if t.PartETags == nil {
		t.PartETags = map[string]string{}
	}
	return &t, nil
}

func (r *UploadTaskRepository) UpdateChunkProgress(ctx context.Context, id uuid.UUID, uploadedChunks int, uploadedSize int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE upload_tasks SET uploaded_chunks = $2, uploaded_size = $3, status = 'uploading', updated_at = NOW()
		 WHERE id = $1 AND status NOT IN ('completed', 'failed')`, id, uploadedChunks, uploadedSize)
	return err
}

// RecordMultipartPart stores a part etag atomically and recomputes progress from etag keys
// so concurrent chunk uploads cannot clobber each other via read-modify-write races.
func (r *UploadTaskRepository) RecordMultipartPart(ctx context.Context, id uuid.UUID, partNumber int32, etag string) error {
	partKey := strconv.Itoa(int(partNumber))
	_, err := r.db.ExecContext(ctx, `
		UPDATE upload_tasks
		SET part_etags = jsonb_set(COALESCE(part_etags, '{}'::jsonb), ARRAY[$2::text], to_jsonb($3::text), true),
		    uploaded_chunks = (
		      SELECT COUNT(*)::int
		      FROM jsonb_object_keys(
		        jsonb_set(COALESCE(part_etags, '{}'::jsonb), ARRAY[$2::text], to_jsonb($3::text), true)
		      )
		    ),
		    uploaded_size = LEAST(
		      total_size,
		      (
		        SELECT COUNT(*)::bigint
		        FROM jsonb_object_keys(
		          jsonb_set(COALESCE(part_etags, '{}'::jsonb), ARRAY[$2::text], to_jsonb($3::text), true)
		        )
		      ) * chunk_size
		    ),
		    status = 'uploading',
		    updated_at = NOW()
		WHERE id = $1
		  AND status NOT IN ('completed', 'failed')`,
		id, partKey, etag,
	)
	return err
}

// UpdateMultipartPart is kept for older call sites; prefers RecordMultipartPart.
func (r *UploadTaskRepository) UpdateMultipartPart(ctx context.Context, id uuid.UUID, partNumber int32, etag string, uploadedChunks int, uploadedSize int64) error {
	_ = uploadedChunks
	_ = uploadedSize
	return r.RecordMultipartPart(ctx, id, partNumber, etag)
}

func (r *UploadTaskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE upload_tasks SET status = $2, updated_at = NOW() WHERE id = $1`, id, status)
	return err
}

// ClaimCompleting transitions a task into the completing state so only one worker finalizes it.
func (r *UploadTaskRepository) ClaimCompleting(ctx context.Context, id uuid.UUID) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE upload_tasks SET status = 'completing', updated_at = NOW()
		 WHERE id = $1 AND status IN ('pending', 'uploading')`, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
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

// TryAcquireChunkLease reserves one global upload slot (multi-instance safe).
// Returns leaseID for ReleaseChunkLease. Expired leases are reclaimed first.
func (r *UploadTaskRepository) TryAcquireChunkLease(ctx context.Context, maxSlots int, holder string, taskID *uuid.UUID, ttl time.Duration) (int64, error) {
	if maxSlots < 1 {
		maxSlots = 1
	}
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM upload_chunk_leases WHERE expires_at < NOW()`); err != nil {
		// Table may not exist yet on old deploys before migrate — surface error.
		return 0, err
	}

	var active int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM upload_chunk_leases`).Scan(&active); err != nil {
		return 0, err
	}
	if active >= maxSlots {
		return 0, nil
	}

	var leaseID int64
	err = tx.QueryRowContext(ctx,
		`INSERT INTO upload_chunk_leases (task_id, holder, expires_at)
		 VALUES ($1, $2, NOW() + ($3 * INTERVAL '1 second'))
		 RETURNING id`,
		taskID, holder, int(ttl.Seconds()),
	).Scan(&leaseID)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return leaseID, nil
}

// ReleaseChunkLease frees a previously acquired upload slot.
func (r *UploadTaskRepository) ReleaseChunkLease(ctx context.Context, leaseID int64) error {
	if leaseID <= 0 {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM upload_chunk_leases WHERE id = $1`, leaseID)
	return err
}
