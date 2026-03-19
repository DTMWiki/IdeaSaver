package repository

import (
	"context"
	"database/sql"
	"strings"
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
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, parent_id, name, storage_key, is_directory, mime_type, size,
		        public_url, thumbnail_key, moderation_status, moderation_reason, moderated_by, moderated_at,
		        deleted_at, created_at, updated_at
		 FROM files WHERE id = $1`, id)
	return scanFile(row.Scan)
}

func (r *FileRepository) ListByParent(ctx context.Context, userID uuid.UUID, parentID *uuid.UUID) ([]model.File, error) {
	var rows *sql.Rows
	var err error

	if parentID == nil {
		rows, err = r.db.QueryContext(ctx,
			`SELECT id, user_id, parent_id, name, storage_key, is_directory, mime_type, size,
			        public_url, thumbnail_key, moderation_status, moderation_reason, moderated_by, moderated_at,
			        deleted_at, created_at, updated_at
			 FROM files WHERE user_id = $1 AND parent_id IS NULL AND deleted_at IS NULL
			 ORDER BY is_directory DESC, name ASC`, userID)
	} else {
		rows, err = r.db.QueryContext(ctx,
			`SELECT id, user_id, parent_id, name, storage_key, is_directory, mime_type, size,
			        public_url, thumbnail_key, moderation_status, moderation_reason, moderated_by, moderated_at,
			        deleted_at, created_at, updated_at
			 FROM files WHERE user_id = $1 AND parent_id = $2 AND deleted_at IS NULL
			 ORDER BY is_directory DESC, name ASC`, userID, parentID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []model.File
	for rows.Next() {
		f, err := scanFile(rows.Scan)
		if err != nil {
			return nil, err
		}
		files = append(files, *f)
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
		        public_url, thumbnail_key, moderation_status, moderation_reason, moderated_by, moderated_at,
		        deleted_at, created_at, updated_at
		 FROM files WHERE user_id = $1 AND deleted_at IS NOT NULL
		 ORDER BY deleted_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []model.File
	for rows.Next() {
		f, err := scanFile(rows.Scan)
		if err != nil {
			return nil, err
		}
		files = append(files, *f)
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
		        public_url, thumbnail_key, moderation_status, moderation_reason, moderated_by, moderated_at,
		        deleted_at, created_at, updated_at
		 FROM files WHERE deleted_at IS NULL
		 ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var files []model.File
	for rows.Next() {
		f, err := scanFile(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		files = append(files, *f)
	}
	return files, total, nil
}

// FindByStorageKey finds a file by its OSS storage key.
func (r *FileRepository) FindByStorageKey(ctx context.Context, storageKey string) (*model.File, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, parent_id, name, storage_key, is_directory, mime_type, size,
		        public_url, thumbnail_key, moderation_status, moderation_reason, moderated_by, moderated_at,
		        deleted_at, created_at, updated_at
		 FROM files WHERE storage_key = $1 AND deleted_at IS NULL`, storageKey)
	return scanFile(row.Scan)
}

func (r *FileRepository) UpdateModeration(ctx context.Context, id uuid.UUID, status, reason string, adminID *uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE files
		 SET moderation_status = $2,
		     moderation_reason = $3,
		     moderated_by = $4,
		     moderated_at = NOW(),
		     updated_at = NOW()
		 WHERE id = $1`,
		id, status, reason, adminID,
	)
	return err
}

func (r *FileRepository) ExistsByName(ctx context.Context, userID uuid.UUID, parentID *uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return false, nil
	}

	query := `SELECT EXISTS(
		SELECT 1
		FROM files
		WHERE user_id = $1
		  AND (($2::uuid IS NULL AND parent_id IS NULL) OR parent_id = $2)
		  AND name = $3
		  AND deleted_at IS NULL
		  AND ($4::uuid IS NULL OR id <> $4)
	)`

	var exists bool
	if err := r.db.QueryRowContext(ctx, query, userID, parentID, name, excludeID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func scanFile(scan func(dest ...any) error) (*model.File, error) {
	var f model.File
	var parentID sql.NullString
	var storageKey sql.NullString
	var mimeType sql.NullString
	var publicURL sql.NullString
	var thumbnailKey sql.NullString
	var moderationStatus sql.NullString
	var moderationReason sql.NullString
	var moderatedBy sql.NullString
	var moderatedAt sql.NullTime
	var deletedAt sql.NullTime

	if err := scan(
		&f.ID,
		&f.UserID,
		&parentID,
		&f.Name,
		&storageKey,
		&f.IsDirectory,
		&mimeType,
		&f.Size,
		&publicURL,
		&thumbnailKey,
		&moderationStatus,
		&moderationReason,
		&moderatedBy,
		&moderatedAt,
		&deletedAt,
		&f.CreatedAt,
		&f.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if parentID.Valid {
		if parsed, err := uuid.Parse(parentID.String); err == nil {
			f.ParentID = &parsed
		}
	}
	f.StorageKey = storageKey.String
	f.MimeType = mimeType.String
	f.PublicURL = publicURL.String
	f.ThumbnailKey = thumbnailKey.String
	if moderationStatus.Valid {
		f.ModerationStatus = moderationStatus.String
	} else {
		f.ModerationStatus = "normal"
	}
	f.ModerationReason = moderationReason.String
	if moderatedBy.Valid {
		if parsed, err := uuid.Parse(moderatedBy.String); err == nil {
			f.ModeratedBy = &parsed
		}
	}
	if moderatedAt.Valid {
		value := moderatedAt.Time
		f.ModeratedAt = &value
	}
	if deletedAt.Valid {
		value := deletedAt.Time
		f.DeletedAt = &value
	}

	return &f, nil
}
