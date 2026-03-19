package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/DTMWiki/IdeaSaver/server/internal/model"
	"github.com/DTMWiki/IdeaSaver/server/internal/requestctx"
	"github.com/google/uuid"
)

// AuditLogRepository handles audit log database operations.
type AuditLogRepository struct {
	db *sql.DB
}

type AuditLogFilter struct {
	Action  string
	User    string
	Keyword string
	StartAt *time.Time
	EndAt   *time.Time
}

func NewAuditLogRepository(db *sql.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(ctx context.Context, log *model.AuditLog) error {
	detailsJSON, _ := json.Marshal(log.Details)
	meta := requestctx.GetMeta(ctx)
	var ipAddress any
	if value := firstNonEmptyTrim(log.IPAddress, meta.IPAddress); value != "" {
		ipAddress = value
	}
	var userAgent any
	if value := firstNonEmptyTrim(log.UserAgent, meta.UserAgent); value != "" {
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
func (r *AuditLogRepository) ListAll(ctx context.Context, filter AuditLogFilter, offset, limit int) ([]model.AuditLog, int, error) {
	whereClause, args := buildAuditLogWhere(filter)

	var total int
	countQuery := `SELECT COUNT(*) FROM audit_logs al LEFT JOIN users u ON al.user_id = u.id` + whereClause
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listArgs := append(append([]any{}, args...), limit, offset)
	query := `SELECT al.id, al.user_id, al.action, al.resource, al.resource_id, al.details,
		        al.ip_address, al.user_agent, al.created_at, u.username
		 FROM audit_logs al LEFT JOIN users u ON al.user_id = u.id` +
		whereClause +
		` ORDER BY al.created_at DESC LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)

	rows, err := r.db.QueryContext(ctx, query, listArgs...)
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

func firstNonEmptyTrim(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func buildAuditLogWhere(filter AuditLogFilter) (string, []any) {
	var clauses []string
	var args []any

	if value := strings.TrimSpace(filter.Action); value != "" {
		args = append(args, value)
		clauses = append(clauses, "al.action = $"+strconv.Itoa(len(args)))
	}
	if value := strings.TrimSpace(filter.User); value != "" {
		args = append(args, "%"+value+"%")
		placeholder := "$" + strconv.Itoa(len(args))
		clauses = append(clauses, "(u.username ILIKE "+placeholder+" OR CAST(al.user_id AS TEXT) ILIKE "+placeholder+")")
	}
	if value := strings.TrimSpace(filter.Keyword); value != "" {
		args = append(args, "%"+value+"%")
		placeholder := "$" + strconv.Itoa(len(args))
		clauses = append(clauses, "(al.action ILIKE "+placeholder+" OR COALESCE(al.resource, '') ILIKE "+placeholder+" OR COALESCE(al.details::text, '') ILIKE "+placeholder+" OR COALESCE(u.username, '') ILIKE "+placeholder+")")
	}
	if filter.StartAt != nil {
		args = append(args, *filter.StartAt)
		clauses = append(clauses, "al.created_at >= $"+strconv.Itoa(len(args)))
	}
	if filter.EndAt != nil {
		args = append(args, *filter.EndAt)
		clauses = append(clauses, "al.created_at <= $"+strconv.Itoa(len(args)))
	}

	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}
