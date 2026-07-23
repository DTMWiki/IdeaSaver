package repository

import (
	"context"
	"database/sql"
	"fmt"
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
		`UPDATE files SET deleted_at = $2, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id, now)
	return err
}

// SoftDeleteSubtree soft-deletes id and all descendants (same user), returning total file size freed.
func (r *FileRepository) SoftDeleteSubtree(ctx context.Context, rootID, userID uuid.UUID) (int64, error) {
	now := time.Now()
	var freed sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
		WITH RECURSIVE tree AS (
			SELECT id, is_directory, size
			FROM files
			WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
			UNION ALL
			SELECT f.id, f.is_directory, f.size
			FROM files f
			INNER JOIN tree t ON f.parent_id = t.id
			WHERE f.user_id = $2 AND f.deleted_at IS NULL
		),
		upd AS (
			UPDATE files
			SET deleted_at = $3, updated_at = NOW()
			WHERE id IN (SELECT id FROM tree) AND deleted_at IS NULL
			RETURNING id, is_directory, size
		)
		SELECT COALESCE(SUM(size), 0)
		FROM upd
		WHERE is_directory = FALSE
	`, rootID, userID, now).Scan(&freed)
	if err != nil {
		return 0, err
	}
	if freed.Valid {
		return freed.Int64, nil
	}
	return 0, nil
}

func (r *FileRepository) Restore(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE files SET deleted_at = NULL, updated_at = NOW() WHERE id = $1`, id)
	return err
}

// RestoreSubtree clears deleted_at for root + descendants, optionally renaming the root.
func (r *FileRepository) RestoreSubtree(ctx context.Context, rootID, userID uuid.UUID, rootName string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	rootName = strings.TrimSpace(rootName)
	if rootName != "" {
		if _, err := tx.ExecContext(ctx,
			`UPDATE files SET name = $2, updated_at = NOW() WHERE id = $1 AND user_id = $3`,
			rootID, rootName, userID,
		); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `
		WITH RECURSIVE tree AS (
			SELECT id FROM files WHERE id = $1 AND user_id = $2
			UNION ALL
			SELECT f.id FROM files f
			INNER JOIN tree t ON f.parent_id = t.id
			WHERE f.user_id = $2
		)
		UPDATE files
		SET deleted_at = NULL, updated_at = NOW()
		WHERE id IN (SELECT id FROM tree) AND deleted_at IS NOT NULL
	`, rootID, userID); err != nil {
		return err
	}
	return tx.Commit()
}

// ListSubtree returns root + all descendants (any deleted state).
// depthDesc=true orders deepest first (for permanent delete OSS cleanup).
func (r *FileRepository) ListSubtree(ctx context.Context, rootID, userID uuid.UUID) ([]model.File, error) {
	return r.listSubtree(ctx, rootID, userID, true, false)
}

// ListActiveSubtree returns non-deleted root + descendants, parents before children (for copy).
func (r *FileRepository) ListActiveSubtree(ctx context.Context, rootID, userID uuid.UUID) ([]model.File, error) {
	return r.listSubtree(ctx, rootID, userID, false, true)
}

func (r *FileRepository) listSubtree(ctx context.Context, rootID, userID uuid.UUID, depthDesc, activeOnly bool) ([]model.File, error) {
	order := "depth DESC"
	if !depthDesc {
		order = "depth ASC"
	}
	activeFilter := ""
	if activeOnly {
		activeFilter = " AND f.deleted_at IS NULL"
	}
	rootFilter := ""
	if activeOnly {
		rootFilter = " AND deleted_at IS NULL"
	}
	// #nosec G201 -- order/filter are fixed constants, not user input
	q := fmt.Sprintf(`
		WITH RECURSIVE tree AS (
			SELECT id, user_id, parent_id, name, storage_key, is_directory, mime_type, size,
			       public_url, thumbnail_key, moderation_status, moderation_reason, moderated_by, moderated_at,
			       deleted_at, created_at, updated_at, 0 AS depth
			FROM files
			WHERE id = $1 AND user_id = $2%s
			UNION ALL
			SELECT f.id, f.user_id, f.parent_id, f.name, f.storage_key, f.is_directory, f.mime_type, f.size,
			       f.public_url, f.thumbnail_key, f.moderation_status, f.moderation_reason, f.moderated_by, f.moderated_at,
			       f.deleted_at, f.created_at, f.updated_at, t.depth + 1
			FROM files f
			INNER JOIN tree t ON f.parent_id = t.id
			WHERE f.user_id = $2%s
		)
		SELECT id, user_id, parent_id, name, storage_key, is_directory, mime_type, size,
		       public_url, thumbnail_key, moderation_status, moderation_reason, moderated_by, moderated_at,
		       deleted_at, created_at, updated_at
		FROM tree
		ORDER BY %s
	`, rootFilter, activeFilter, order)

	rows, err := r.db.QueryContext(ctx, q, rootID, userID)
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

// ListDeletedSubtree returns soft-deleted descendants that share the same deleted_at wave as root
// (children soft-deleted with the folder in SoftDeleteSubtree).
func (r *FileRepository) ListDeletedSubtree(ctx context.Context, rootID, userID uuid.UUID) ([]model.File, error) {
	return r.ListSubtree(ctx, rootID, userID)
}

// SumActiveSubtreeSize returns size of non-directory files under root that are currently not deleted.
func (r *FileRepository) SumActiveSubtreeSize(ctx context.Context, rootID, userID uuid.UUID) (int64, error) {
	var total sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
		WITH RECURSIVE tree AS (
			SELECT id FROM files WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
			UNION ALL
			SELECT f.id FROM files f
			INNER JOIN tree t ON f.parent_id = t.id
			WHERE f.user_id = $2 AND f.deleted_at IS NULL
		)
		SELECT COALESCE(SUM(size), 0)
		FROM files
		WHERE id IN (SELECT id FROM tree)
		  AND is_directory = FALSE
		  AND deleted_at IS NULL
	`, rootID, userID).Scan(&total)
	if err != nil {
		return 0, err
	}
	if total.Valid {
		return total.Int64, nil
	}
	return 0, nil
}

// SumDeletedSubtreeSize returns size of soft-deleted files in subtree (for restore quota).
func (r *FileRepository) SumDeletedSubtreeSize(ctx context.Context, rootID, userID uuid.UUID) (int64, error) {
	var total sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
		WITH RECURSIVE tree AS (
			SELECT id FROM files WHERE id = $1 AND user_id = $2
			UNION ALL
			SELECT f.id FROM files f
			INNER JOIN tree t ON f.parent_id = t.id
			WHERE f.user_id = $2
		)
		SELECT COALESCE(SUM(size), 0)
		FROM files
		WHERE id IN (SELECT id FROM tree)
		  AND is_directory = FALSE
		  AND deleted_at IS NOT NULL
	`, rootID, userID).Scan(&total)
	if err != nil {
		return 0, err
	}
	if total.Valid {
		return total.Int64, nil
	}
	return 0, nil
}

// IsAncestor reports whether ancestorID is the same as or an ancestor of nodeID.
func (r *FileRepository) IsAncestor(ctx context.Context, ancestorID, nodeID, userID uuid.UUID) (bool, error) {
	if ancestorID == nodeID {
		return true, nil
	}
	var ok bool
	err := r.db.QueryRowContext(ctx, `
		WITH RECURSIVE up AS (
			SELECT id, parent_id FROM files WHERE id = $1 AND user_id = $3
			UNION ALL
			SELECT f.id, f.parent_id FROM files f
			INNER JOIN up u ON f.id = u.parent_id
			WHERE f.user_id = $3
		)
		SELECT EXISTS(SELECT 1 FROM up WHERE id = $2)
	`, nodeID, ancestorID, userID).Scan(&ok)
	return ok, err
}

func (r *FileRepository) PermanentDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM files WHERE id = $1`, id)
	return err
}

// PermanentDeleteSubtree deletes the whole tree (leaves first via CASCADE or single root delete).
func (r *FileRepository) PermanentDeleteSubtree(ctx context.Context, rootID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM files WHERE id = $1 AND user_id = $2`, rootID, userID)
	return err
}

func (r *FileRepository) ListTrash(ctx context.Context, userID uuid.UUID) ([]model.File, error) {
	// Only show trash roots: skip children that were soft-deleted with a parent folder.
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, parent_id, name, storage_key, is_directory, mime_type, size,
		        public_url, thumbnail_key, moderation_status, moderation_reason, moderated_by, moderated_at,
		        deleted_at, created_at, updated_at
		 FROM files f
		 WHERE f.user_id = $1
		   AND f.deleted_at IS NOT NULL
		   AND (
		     f.parent_id IS NULL
		     OR NOT EXISTS (
		       SELECT 1 FROM files p
		       WHERE p.id = f.parent_id
		         AND p.user_id = $1
		         AND p.deleted_at IS NOT NULL
		     )
		   )
		 ORDER BY f.deleted_at DESC`, userID)
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
	// Prefer FileService.CleanupTrash which also removes OSS objects.
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM files WHERE deleted_at IS NOT NULL AND deleted_at < NOW() - INTERVAL '1 day' * $1`,
		retentionDays)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ListExpiredTrash returns soft-deleted files past retention for cleanup (OSS + DB).
func (r *FileRepository) ListExpiredTrash(ctx context.Context, retentionDays int) ([]model.File, error) {
	if retentionDays < 1 {
		retentionDays = 1
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, parent_id, name, storage_key, is_directory, mime_type, size,
		        public_url, thumbnail_key, moderation_status, moderation_reason, moderated_by, moderated_at,
		        deleted_at, created_at, updated_at
		 FROM files
		 WHERE deleted_at IS NOT NULL
		   AND deleted_at < NOW() - INTERVAL '1 day' * $1
		 ORDER BY deleted_at ASC
		 LIMIT 500`,
		retentionDays)
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

// ListAll returns all files across all users (for admin).
func (r *FileRepository) ListAll(ctx context.Context, keyword string, offset, limit int) ([]model.File, int, error) {
	keyword = strings.TrimSpace(keyword)
	countQuery := `SELECT COUNT(*)
		FROM files f
		LEFT JOIN users u ON u.id = f.user_id
		WHERE f.deleted_at IS NULL`
	args := []any{}
	clauses := []string{}
	if keyword != "" {
		args = append(args, "%"+keyword+"%")
		clauses = append(clauses, `(f.name ILIKE $1 OR COALESCE(u.username, '') ILIKE $1 OR COALESCE(f.mime_type, '') ILIKE $1)`)
	}
	if len(clauses) > 0 {
		countQuery += " AND " + strings.Join(clauses, " AND ")
	}

	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	listQuery := `SELECT f.id, f.user_id, f.parent_id, f.name, f.storage_key, f.is_directory, f.mime_type, f.size,
		        f.public_url, f.thumbnail_key, f.moderation_status, f.moderation_reason, f.moderated_by, f.moderated_at,
		        f.deleted_at, f.created_at, f.updated_at, COALESCE(u.username, '')
		 FROM files f
		 LEFT JOIN users u ON u.id = f.user_id
		 WHERE f.deleted_at IS NULL`
	if len(clauses) > 0 {
		listQuery += " AND " + strings.Join(clauses, " AND ")
	}
	listQuery += fmt.Sprintf(" ORDER BY f.created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var files []model.File
	for rows.Next() {
		f, err := scanFileWithUsername(rows.Scan)
		if err != nil {
			return nil, 0, err
		}
		files = append(files, *f)
	}
	return files, total, nil
}

func scanFileWithUsername(scan func(dest ...any) error) (*model.File, error) {
	file, err := scanFileWithExtras(scan, true)
	if err != nil {
		return nil, err
	}
	return file, nil
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
	return scanFileWithExtras(scan, false)
}

func scanFileWithExtras(scan func(dest ...any) error, withUsername bool) (*model.File, error) {
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
	var username sql.NullString

	scanArgs := []any{
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
	}
	if withUsername {
		scanArgs = append(scanArgs, &username)
	}

	if err := scan(scanArgs...); err != nil {
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
	f.Username = username.String

	return &f, nil
}
