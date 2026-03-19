package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/DTMWiki/IdeaSaver/server/internal/model"
	"github.com/google/uuid"
)

func TestCreateTreatsEmptyIPAndUserAgentAsNull(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	repo := NewAuditLogRepository(db)
	userID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO audit_logs (user_id, action, resource, resource_id, details, ip_address, user_agent)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, created_at`)).
		WithArgs(userID, "upload", "file", nil, []byte(`{"file_name":"cover.png"}`), nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(1, now))

	log := &model.AuditLog{
		UserID:   userID,
		Action:   "upload",
		Resource: "file",
		Details: map[string]any{
			"file_name": "cover.png",
		},
	}

	if err := repo.Create(context.Background(), log); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestListByUserHandlesNullableFields(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	repo := NewAuditLogRepository(db)
	userID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM audit_logs WHERE user_id = $1`)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT al.id, al.user_id, al.action, al.resource, al.resource_id, al.details,
		        al.ip_address, al.user_agent, al.created_at, u.username
		 FROM audit_logs al LEFT JOIN users u ON al.user_id = u.id
		 WHERE al.user_id = $1
		 ORDER BY al.created_at DESC LIMIT $2 OFFSET $3`)).
		WithArgs(userID, 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "action", "resource", "resource_id", "details",
			"ip_address", "user_agent", "created_at", "username",
		}).AddRow(
			1,
			userID.String(),
			"upload",
			nil,
			nil,
			[]byte(`{"file_name":"cover.png"}`),
			nil,
			nil,
			now,
			nil,
		))

	logs, total, err := repo.ListByUser(context.Background(), userID, 0, 20)
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if total != 1 {
		t.Fatalf("ListByUser() total = %d, want 1", total)
	}
	if len(logs) != 1 {
		t.Fatalf("ListByUser() len = %d, want 1", len(logs))
	}
	if logs[0].Resource != "" {
		t.Fatalf("Resource = %q, want empty", logs[0].Resource)
	}
	if logs[0].IPAddress != "" {
		t.Fatalf("IPAddress = %q, want empty", logs[0].IPAddress)
	}
	if logs[0].UserAgent != "" {
		t.Fatalf("UserAgent = %q, want empty", logs[0].UserAgent)
	}
	if logs[0].Username != "" {
		t.Fatalf("Username = %q, want empty", logs[0].Username)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
