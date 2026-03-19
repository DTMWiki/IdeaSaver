package service

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/DTMWiki/IdeaSaver/server/internal/repository"
	"github.com/google/uuid"
)

func newMockRepos(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *repository.Repositories) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	return db, mock, repository.NewRepositories(db)
}

func makeFileRows(fileID, userID uuid.UUID, status, reason string) *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows([]string{
		"id", "user_id", "parent_id", "name", "storage_key", "is_directory", "mime_type", "size",
		"public_url", "thumbnail_key", "moderation_status", "moderation_reason", "moderated_by", "moderated_at",
		"deleted_at", "created_at", "updated_at",
	}).AddRow(
		fileID, userID, nil, "example.png", userID.String()+"/example.png", false, "image/png", int64(1234),
		"https://example/s/"+userID.String()+"/example.png", "", status, reason, nil, nil,
		nil, now, now,
	)
}

func TestFileServiceSubmitAppealSuccess(t *testing.T) {
	db, mock, repos := newMockRepos(t)
	defer db.Close()

	svc := NewFileService(&config.Config{}, repos, nil)
	fileID := uuid.New()
	userID := uuid.New()
	appealID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`FROM files WHERE id = \$1`).
		WithArgs(fileID).
		WillReturnRows(makeFileRows(fileID, userID, "banned", "违规图片"))

	mock.ExpectQuery(`FROM file_appeals\s+WHERE file_id = \$1 AND status = 'pending'`).
		WithArgs(fileID).
		WillReturnError(sql.ErrNoRows)

	mock.ExpectQuery(`INSERT INTO file_appeals`).
		WithArgs(fileID, userID, "pending", "已整改，申请复审").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(appealID, now, now))

	mock.ExpectQuery(`INSERT INTO audit_logs`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(1), now))

	appeal, err := svc.SubmitAppeal(context.Background(), fileID, userID, "已整改，申请复审")
	if err != nil {
		t.Fatalf("SubmitAppeal returned error: %v", err)
	}
	if appeal.ID != appealID {
		t.Fatalf("unexpected appeal id: got %s want %s", appeal.ID, appealID)
	}
	if appeal.Status != "pending" {
		t.Fatalf("unexpected appeal status: %s", appeal.Status)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestFileServiceSubmitAppealRejectsDuplicatePending(t *testing.T) {
	db, mock, repos := newMockRepos(t)
	defer db.Close()

	svc := NewFileService(&config.Config{}, repos, nil)
	fileID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`FROM files WHERE id = \$1`).
		WithArgs(fileID).
		WillReturnRows(makeFileRows(fileID, userID, "banned", "违规图片"))

	mock.ExpectQuery(`FROM file_appeals\s+WHERE file_id = \$1 AND status = 'pending'`).
		WithArgs(fileID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "file_id", "user_id", "status", "reason", "admin_comment", "reviewed_by", "reviewed_at", "created_at", "updated_at",
		}).AddRow(uuid.New(), fileID, userID, "pending", "重复申诉", "", nil, nil, now, now))

	_, err := svc.SubmitAppeal(context.Background(), fileID, userID, "再次提交")
	if err == nil || !strings.Contains(err.Error(), "已有待处理申诉") {
		t.Fatalf("expected duplicate pending error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestShareServiceDeleteSharePermissionDenied(t *testing.T) {
	db, mock, repos := newMockRepos(t)
	defer db.Close()

	svc := NewShareService(&config.Config{}, repos, nil)
	shareID := uuid.New()
	ownerID := uuid.New()
	callerID := uuid.New()
	fileID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`FROM shares WHERE id = \$1`).
		WithArgs(shareID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "file_id", "code", "password", "expires_at", "view_count", "created_at",
		}).AddRow(shareID, ownerID, fileID, "abcdef", "", nil, 0, now))

	err := svc.DeleteShare(context.Background(), shareID, callerID)
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("expected permission denied, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestShareServiceAccessShareBlockedWhenFileBanned(t *testing.T) {
	db, mock, repos := newMockRepos(t)
	defer db.Close()

	svc := NewShareService(&config.Config{}, repos, nil)
	shareID := uuid.New()
	fileID := uuid.New()
	ownerID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`FROM shares WHERE code = \$1`).
		WithArgs("public-code").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "file_id", "code", "password", "expires_at", "view_count", "created_at",
		}).AddRow(shareID, ownerID, fileID, "public-code", "", nil, 0, now))

	mock.ExpectExec(`UPDATE shares SET view_count = view_count \+ 1 WHERE id = \$1`).
		WithArgs(shareID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery(`FROM files WHERE id = \$1`).
		WithArgs(fileID).
		WillReturnRows(makeFileRows(fileID, ownerID, "banned", "违规资源"))

	_, err := svc.AccessShare(context.Background(), "public-code", "")
	if err == nil || !strings.Contains(err.Error(), "已被封禁") {
		t.Fatalf("expected banned error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
