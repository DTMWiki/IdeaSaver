package repository

import (
	"context"
	"database/sql"

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
		`SELECT id, username, display_name, email, role, storage_quota, storage_used, created_at, updated_at
		 FROM users WHERE username = $1`, username).Scan(
		&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.Role,
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
		`SELECT id, username, display_name, email, role, storage_quota, storage_used, created_at, updated_at
		 FROM users WHERE id = $1`, id).Scan(
		&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.Role,
		&u.StorageQuota, &u.StorageUsed, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Upsert(ctx context.Context, u *model.User) error {
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
		`UPDATE users SET storage_used = storage_used + $2, updated_at = NOW() WHERE id = $1`,
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
		`SELECT id, username, display_name, email, role, storage_quota, storage_used, created_at, updated_at
		 FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.Role,
			&u.StorageQuota, &u.StorageUsed, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, nil
}
