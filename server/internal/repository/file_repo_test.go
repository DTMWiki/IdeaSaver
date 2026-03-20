package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestFindByStorageKeyHandlesNullableFields(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	repo := NewFileRepository(db)
	fileID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, parent_id, name, storage_key, is_directory, mime_type, size,
		        public_url, thumbnail_key, moderation_status, moderation_reason, moderated_by, moderated_at,
		        deleted_at, created_at, updated_at
		 FROM files WHERE storage_key = $1 AND deleted_at IS NULL`)).
		WithArgs("user/random.png").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "parent_id", "name", "storage_key", "is_directory", "mime_type", "size",
			"public_url", "thumbnail_key", "moderation_status", "moderation_reason", "moderated_by", "moderated_at",
			"deleted_at", "created_at", "updated_at",
		}).AddRow(
			fileID.String(),
			userID.String(),
			nil,
			"cover.png",
			"user/random.png",
			false,
			"image/png",
			int64(123),
			"https://example.com/s/user/random.png",
			nil,
			"normal",
			nil,
			nil,
			nil,
			nil,
			now,
			now,
		))

	file, err := repo.FindByStorageKey(context.Background(), "user/random.png")
	if err != nil {
		t.Fatalf("FindByStorageKey() error = %v", err)
	}
	if file == nil {
		t.Fatal("FindByStorageKey() returned nil file")
	}
	if file.ThumbnailKey != "" {
		t.Fatalf("ThumbnailKey = %q, want empty", file.ThumbnailKey)
	}
	if file.ModerationReason != "" {
		t.Fatalf("ModerationReason = %q, want empty", file.ModerationReason)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestListByParentHandlesNullableDirectoryFields(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	repo := NewFileRepository(db)
	userID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, parent_id, name, storage_key, is_directory, mime_type, size,
		        public_url, thumbnail_key, moderation_status, moderation_reason, moderated_by, moderated_at,
		        deleted_at, created_at, updated_at
		 FROM files WHERE user_id = $1 AND parent_id IS NULL AND deleted_at IS NULL
		 ORDER BY is_directory DESC, name ASC`)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "parent_id", "name", "storage_key", "is_directory", "mime_type", "size",
			"public_url", "thumbnail_key", "moderation_status", "moderation_reason", "moderated_by", "moderated_at",
			"deleted_at", "created_at", "updated_at",
		}).AddRow(
			uuid.New().String(),
			userID.String(),
			nil,
			"Documents",
			nil,
			true,
			nil,
			int64(0),
			nil,
			nil,
			"normal",
			nil,
			nil,
			nil,
			nil,
			now,
			now,
		))

	files, err := repo.ListByParent(context.Background(), userID, nil)
	if err != nil {
		t.Fatalf("ListByParent() error = %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("ListByParent() len = %d, want 1", len(files))
	}
	if files[0].StorageKey != "" {
		t.Fatalf("StorageKey = %q, want empty", files[0].StorageKey)
	}
	if files[0].MimeType != "" {
		t.Fatalf("MimeType = %q, want empty", files[0].MimeType)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
