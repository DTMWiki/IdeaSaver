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
