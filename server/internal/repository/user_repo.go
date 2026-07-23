package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/DTMWiki/IdeaSaver/server/internal/model"
	"github.com/google/uuid"
)

// UserRepository handles user database operations.
type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var u model.User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, username, display_name, email, role, COALESCE(oidc_sub, ''), storage_quota, storage_used, created_at, updated_at
		 FROM users WHERE username = $1`, username).Scan(
		&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.Role, &u.OIDCSub,
		&u.StorageQuota, &u.StorageUsed, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByOIDCSub(ctx context.Context, sub string) (*model.User, error) {
	sub = strings.TrimSpace(sub)
	if sub == "" {
		return nil, sql.ErrNoRows
	}
	var u model.User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, username, display_name, email, role, COALESCE(oidc_sub, ''), storage_quota, storage_used, created_at, updated_at
		 FROM users WHERE oidc_sub = $1`, sub).Scan(
		&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.Role, &u.OIDCSub,
		&u.StorageQuota, &u.StorageUsed, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var u model.User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, username, display_name, email, role, COALESCE(oidc_sub, ''), storage_quota, storage_used, created_at, updated_at
		 FROM users WHERE id = $1`, id).Scan(
		&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.Role, &u.OIDCSub,
		&u.StorageQuota, &u.StorageUsed, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpsertByOIDC links an IdP subject to a local user.
// Identity is keyed by oidc_sub when available; username may change safely.
func (r *UserRepository) UpsertByOIDC(ctx context.Context, u *model.User) error {
	sub := strings.TrimSpace(u.OIDCSub)
	username := strings.TrimSpace(u.Username)
	if sub == "" {
		return fmt.Errorf("missing oidc subject")
	}
	if username == "" {
		username = "user-" + shortID(sub)
		u.Username = username
	}

	// 1) Prefer stable subject match.
	existing, err := r.FindByOIDCSub(ctx, sub)
	if err == nil {
		return r.updateProfile(ctx, existing.ID, u)
	}
	if err != sql.ErrNoRows {
		return err
	}

	// 2) Legacy migrate: same username, empty oidc_sub → attach subject.
	byName, err := r.FindByUsername(ctx, username)
	if err == nil && strings.TrimSpace(byName.OIDCSub) == "" {
		_, err = r.db.ExecContext(ctx,
			`UPDATE users SET oidc_sub = $2, display_name = $3, email = $4, role = $5, updated_at = NOW()
			 WHERE id = $1`,
			byName.ID, sub, u.DisplayName, u.Email, u.Role,
		)
		if err != nil {
			return err
		}
		u.ID = byName.ID
		u.CreatedAt = byName.CreatedAt
		return nil
	}
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	// Username taken by another subject → derive a unique username.
	if err == nil && strings.TrimSpace(byName.OIDCSub) != "" && byName.OIDCSub != sub {
		username = uniqueUsername(username, sub)
		u.Username = username
	}

	// 3) Insert new user.
	return r.db.QueryRowContext(ctx,
		`INSERT INTO users (username, display_name, email, role, storage_quota, oidc_sub)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at, updated_at`,
		username, u.DisplayName, u.Email, u.Role, u.StorageQuota, sub,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (r *UserRepository) updateProfile(ctx context.Context, id uuid.UUID, u *model.User) error {
	// Keep username unique: only update username when free or already ours.
	_, err := r.db.ExecContext(ctx, `
		UPDATE users SET
		  username = CASE
		    WHEN $2 <> '' AND NOT EXISTS (
		      SELECT 1 FROM users o WHERE o.username = $2 AND o.id <> $1
		    ) THEN $2
		    ELSE username
		  END,
		  display_name = $3,
		  email = $4,
		  role = $5,
		  oidc_sub = COALESCE(NULLIF(oidc_sub, ''), $6),
		  updated_at = NOW()
		WHERE id = $1`,
		id, strings.TrimSpace(u.Username), u.DisplayName, u.Email, u.Role, strings.TrimSpace(u.OIDCSub),
	)
	if err != nil {
		return err
	}
	u.ID = id
	return nil
}

// Upsert is retained for callers that only have a username (no OIDC sub).
func (r *UserRepository) Upsert(ctx context.Context, u *model.User) error {
	if strings.TrimSpace(u.OIDCSub) != "" {
		return r.UpsertByOIDC(ctx, u)
	}
	return r.db.QueryRowContext(ctx,
		`INSERT INTO users (username, display_name, email, role, storage_quota)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (username) DO UPDATE SET
		   display_name = EXCLUDED.display_name,
		   email = EXCLUDED.email,
		   role = EXCLUDED.role,
		   updated_at = NOW()
		 RETURNING id, created_at, updated_at`,
		u.Username, u.DisplayName, u.Email, u.Role, u.StorageQuota,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (r *UserRepository) UpdateStorageUsed(ctx context.Context, userID uuid.UUID, delta int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET storage_used = GREATEST(0, storage_used + $2), updated_at = NOW() WHERE id = $1`,
		userID, delta,
	)
	return err
}

func (r *UserRepository) UpdateQuota(ctx context.Context, userID uuid.UUID, quota int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET storage_quota = $2, updated_at = NOW() WHERE id = $1`,
		userID, quota,
	)
	return err
}

// RecalcAllStorageUsed recomputes storage_used for every user from files + videos.
// Active files only (not soft-deleted); all video records count toward usage.
func (r *UserRepository) RecalcAllStorageUsed(ctx context.Context) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE users u
		SET storage_used = COALESCE((
			SELECT SUM(f.size)
			FROM files f
			WHERE f.user_id = u.id
			  AND f.deleted_at IS NULL
			  AND f.is_directory = FALSE
		), 0) + COALESCE((
			SELECT SUM(v.size)
			FROM videos v
			WHERE v.user_id = u.id
		), 0),
		updated_at = NOW()
	`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// RecalcStorageUsed recomputes storage_used for a single user.
func (r *UserRepository) RecalcStorageUsed(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users u
		SET storage_used = COALESCE((
			SELECT SUM(f.size)
			FROM files f
			WHERE f.user_id = u.id
			  AND f.deleted_at IS NULL
			  AND f.is_directory = FALSE
		), 0) + COALESCE((
			SELECT SUM(v.size)
			FROM videos v
			WHERE v.user_id = u.id
		), 0),
		updated_at = NOW()
		WHERE u.id = $1
	`, userID)
	return err
}

func (r *UserRepository) List(ctx context.Context, offset, limit int) ([]model.User, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, username, display_name, email, role, COALESCE(oidc_sub, ''), storage_quota, storage_used, created_at, updated_at
		 FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.Role, &u.OIDCSub,
			&u.StorageQuota, &u.StorageUsed, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, nil
}

func shortID(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 8 {
		return s
	}
	return s[len(s)-8:]
}

func uniqueUsername(base, sub string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "user"
	}
	if len(base) > 48 {
		base = base[:48]
	}
	return base + "-" + shortID(sub)
}
