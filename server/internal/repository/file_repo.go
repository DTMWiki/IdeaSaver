package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/DTMWiki/IdeaSaver/server/internal/model"
	"github.com/google/uuid"
)

// FileRepository handles file/directory database operations.
type FileRepository struct {
	db *sql.DB
}

func NewFileRepository(db *sql.DB) *FileRepository {
	return &FileRepository{db: db}
}

func (r *FileRepository) Create(ctx context.Context, f *model.File) error {
	return r.db.QueryRowContext(ctx,
		`INSERT INTO files (user_id, parent_id, name, storage_key, is_directory, mime_type, size, public_url, thumbnail_key)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, created_at, updated_at`,
		f.UserID, f.ParentID, f.Name, f.StorageKey, f.IsDirectory,
		f.MimeType, f.Size, f.PublicURL, f.ThumbnailKey,
	).Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt)
}

func (r *FileRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.File, error) {
	var f model.File
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, parent_id, name, storage_key, is_directory, mime_type, size,
		        public_url, thumbnail_key, deleted_at, created_at, updated_at
		 FROM files WHERE id = $1`, id).Scan(
		&f.ID, &f.UserID, &f.ParentID, &f.Name, &f.StorageKey, &f.IsDirectory,
		&f.MimeType, &f.Size, &f.PublicURL, &f.ThumbnailKey,
		&f.DeletedAt, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *FileRepository) ListByParent(ctx context.Context, userID uuid.UUID, parentID *uuid.UUID) ([]model.File, error) {
	var rows *sql.Rows
	var err error

	if parentID == nil {
		rows, err = r.db.QueryContext(ctx,
			`SELECT id, user_id, parent_id, name, storage_key, is_directory, mime_type, size,
			        public_url, thumbnail_key, deleted_at, created_at, updated_at
			 FROM files WHERE user_id = $1 AND parent_id IS NULL AND deleted_at IS NULL
			 ORDER BY is_directory DESC, name ASC`, userID)
	} else {
		rows, err = r.db.QueryContext(ctx,
			`SELECT id, user_id, parent_id, name, storage_key, is_directory, mime_type, size,
			        public_url, thumbnail_key, deleted_at, created_at, updated_at
			 FROM files WHERE user_id = $1 AND parent_id = $2 AND deleted_at IS NULL
			 ORDER BY is_directory DESC, name ASC`, userID, parentID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []model.File
	for rows.Next() {
		var f model.File
		if err := rows.Scan(&f.ID, &f.UserID, &f.ParentID, &f.Name, &f.StorageKey,
			&f.IsDirectory, &f.MimeType, &f.Size, &f.PublicURL, &f.ThumbnailKey,
			&f.DeletedAt, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}

func (r *FileRepository) Rename(ctx context.Context, id uuid.UUID, newName string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE files SET name = $2, updated_at = NOW() WHERE id = $1`, id, newName)
	return err
}

func (r *FileRepository) Move(ctx context.Context, id uuid.UUID, newParentID *uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE files SET parent_id = $2, updated_at = NOW() WHERE id = $1`, id, newParentID)
	return err
}

func (r *FileRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE files SET deleted_at = $2, updated_at = NOW() WHERE id = $1`, id, now)
	return err
}

func (r *FileRepository) Restore(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE files SET deleted_at = NULL, updated_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *FileRepository) PermanentDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM files WHERE id = $1`, id)
	return err
}

func (r *FileRepository) ListTrash(ctx context.Context, userID uuid.UUID) ([]model.File, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, parent_id, name, storage_key, is_directory, mime_type, size,
		        public_url, thumbnail_key, deleted_at, created_at, updated_at
		 FROM files WHERE user_id = $1 AND deleted_at IS NOT NULL
		 ORDER BY deleted_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []model.File
	for rows.Next() {
		var f model.File
		if err := rows.Scan(&f.ID, &f.UserID, &f.ParentID, &f.Name, &f.StorageKey,
			&f.IsDirectory, &f.MimeType, &f.Size, &f.PublicURL, &f.ThumbnailKey,
			&f.DeletedAt, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}

func (r *FileRepository) CleanupTrash(ctx context.Context, retentionDays int) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM files WHERE deleted_at IS NOT NULL AND deleted_at < NOW() - INTERVAL '1 day' * $1`,
		retentionDays)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ListAll returns all files across all users (for admin).
func (r *FileRepository) ListAll(ctx context.Context, offset, limit int) ([]model.File, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM files WHERE deleted_at IS NULL`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, parent_id, name, storage_key, is_directory, mime_type, size,
		        public_url, thumbnail_key, deleted_at, created_at, updated_at
		 FROM files WHERE deleted_at IS NULL
		 ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var files []model.File
	for rows.Next() {
		var f model.File
		if err := rows.Scan(&f.ID, &f.UserID, &f.ParentID, &f.Name, &f.StorageKey,
			&f.IsDirectory, &f.MimeType, &f.Size, &f.PublicURL, &f.ThumbnailKey,
			&f.DeletedAt, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, 0, err
		}
		files = append(files, f)
	}
	return files, total, nil
}

// FindByStorageKey finds a file by its OSS storage key.
func (r *FileRepository) FindByStorageKey(ctx context.Context, userID, filename string) (*model.File, error) {
	storageKey := userID + "/" + filename
	var f model.File
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, parent_id, name, storage_key, is_directory, mime_type, size,
		        public_url, thumbnail_key, deleted_at, created_at, updated_at
		 FROM files WHERE storage_key = $1 AND deleted_at IS NULL`, storageKey).Scan(
		&f.ID, &f.UserID, &f.ParentID, &f.Name, &f.StorageKey, &f.IsDirectory,
		&f.MimeType, &f.Size, &f.PublicURL, &f.ThumbnailKey,
		&f.DeletedAt, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &f, nil
}
