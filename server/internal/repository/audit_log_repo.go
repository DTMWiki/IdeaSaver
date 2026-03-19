package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/DTMWiki/IdeaSaver/server/internal/model"
	"github.com/google/uuid"
)

// AuditLogRepository handles audit log database operations.
type AuditLogRepository struct {
	db *sql.DB
}

func NewAuditLogRepository(db *sql.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(ctx context.Context, log *model.AuditLog) error {
	detailsJSON, _ := json.Marshal(log.Details)
	var ipAddress any
	if value := strings.TrimSpace(log.IPAddress); value != "" {
		ipAddress = value
	}
	var userAgent any
	if value := strings.TrimSpace(log.UserAgent); value != "" {
		userAgent = value
	}
	return r.db.QueryRowContext(ctx,
		`INSERT INTO audit_logs (user_id, action, resource, resource_id, details, ip_address, user_agent)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, created_at`,
		log.UserID, log.Action, log.Resource, log.ResourceID,
		detailsJSON, ipAddress, userAgent,
	).Scan(&log.ID, &log.CreatedAt)
}

// ListByUser returns audit logs for a specific user.
func (r *AuditLogRepository) ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]model.AuditLog, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM audit_logs WHERE user_id = $1`, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT al.id, al.user_id, al.action, al.resource, al.resource_id, al.details,
		        al.ip_address, al.user_agent, al.created_at, u.username
		 FROM audit_logs al LEFT JOIN users u ON al.user_id = u.id
		 WHERE al.user_id = $1
		 ORDER BY al.created_at DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	return scanAuditLogs(rows, total)
}

// ListAll returns all audit logs (for admin).
func (r *AuditLogRepository) ListAll(ctx context.Context, action string, offset, limit int) ([]model.AuditLog, int, error) {
	var total int
	var countQuery string
	var countArgs []any

	if action != "" {
		countQuery = `SELECT COUNT(*) FROM audit_logs WHERE action = $1`
		countArgs = append(countArgs, action)
	} else {
		countQuery = `SELECT COUNT(*) FROM audit_logs`
	}

	err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	var rows *sql.Rows
	if action != "" {
		rows, err = r.db.QueryContext(ctx,
			`SELECT al.id, al.user_id, al.action, al.resource, al.resource_id, al.details,
			        al.ip_address, al.user_agent, al.created_at, u.username
			 FROM audit_logs al LEFT JOIN users u ON al.user_id = u.id
			 WHERE al.action = $1
			 ORDER BY al.created_at DESC LIMIT $2 OFFSET $3`, action, limit, offset)
	} else {
		rows, err = r.db.QueryContext(ctx,
			`SELECT al.id, al.user_id, al.action, al.resource, al.resource_id, al.details,
			        al.ip_address, al.user_agent, al.created_at, u.username
			 FROM audit_logs al LEFT JOIN users u ON al.user_id = u.id
			 ORDER BY al.created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	return scanAuditLogs(rows, total)
}

func scanAuditLogs(rows *sql.Rows, total int) ([]model.AuditLog, int, error) {
	var logs []model.AuditLog
	for rows.Next() {
		var l model.AuditLog
		var detailsJSON []byte
		var userID sql.NullString
		var resource sql.NullString
		var resourceID sql.NullString
		var ipAddress sql.NullString
		var userAgent sql.NullString
		var username sql.NullString
		if err := rows.Scan(&l.ID, &userID, &l.Action, &resource, &resourceID,
			&detailsJSON, &ipAddress, &userAgent, &l.CreatedAt, &username); err != nil {
			return nil, 0, err
		}
		if userID.Valid {
			if parsed, err := uuid.Parse(userID.String); err == nil {
				l.UserID = parsed
			}
		}
		l.Resource = resource.String
		l.IPAddress = ipAddress.String
		l.UserAgent = userAgent.String
		l.Username = username.String
		if resourceID.Valid {
			if parsed, err := uuid.Parse(resourceID.String); err == nil {
				l.ResourceID = &parsed
			}
		}
		if detailsJSON != nil {
			_ = json.Unmarshal(detailsJSON, &l.Details)
		}
		logs = append(logs, l)
	}
	return logs, total, nil
}
