package repository

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"sort"
	"strings"

	_ "github.com/lib/pq"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// NewDB creates a new database connection.
func NewDB(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	log.Println("Database connected successfully")
	return db, nil
}

// Migrate runs pending SQL migration files once, tracked in schema_migrations.
func Migrate(db *sql.DB) error {
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("failed to ensure schema_migrations: %w", err)
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	applied := map[string]bool{}
	rows, err := db.QueryContext(ctx, `SELECT filename FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("failed to list applied migrations: %w", err)
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return err
		}
		applied[name] = true
	}
	rows.Close()

	// Bootstrap: if schema already has core tables but no migration history,
	// mark existing SQL files as applied without re-executing (avoids re-run failures).
	if len(applied) == 0 {
		var usersExist bool
		if err := db.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'users')`,
		).Scan(&usersExist); err != nil {
			return err
		}
		if usersExist {
			for _, name := range names {
				if _, err := db.ExecContext(ctx,
					`INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT DO NOTHING`, name,
				); err != nil {
					return fmt.Errorf("failed to bootstrap migration %s: %w", name, err)
				}
				applied[name] = true
				log.Printf("Bootstrapped migration history: %s", name)
			}
			return nil
		}
	}

	for _, name := range names {
		if applied[name] {
			log.Printf("Skip applied migration: %s", name)
			continue
		}
		content, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", name, err)
		}
		log.Printf("Running migration: %s", name)
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schema_migrations (filename) VALUES ($1)`, name,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

// ListMigrationStatus returns applied migration filenames and pending ones.
func ListMigrationStatus(db *sql.DB) (applied, pending []string, err error) {
	ctx := context.Background()
	_, _ = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return nil, nil, err
	}
	all := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			all = append(all, e.Name())
		}
	}
	sort.Strings(all)

	seen := map[string]bool{}
	rows, err := db.QueryContext(ctx, `SELECT filename FROM schema_migrations ORDER BY filename`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err == nil {
				applied = append(applied, name)
				seen[name] = true
			}
		}
	}
	for _, name := range all {
		if !seen[name] {
			pending = append(pending, name)
		}
	}
	return applied, pending, nil
}

// Repositories holds all repository instances.
type Repositories struct {
	Users       *UserRepository
	Files       *FileRepository
	FileAppeals *FileAppealRepository
	Videos      *VideoRepository
	Shares      *ShareRepository
	UploadTasks *UploadTaskRepository
	AuditLogs   *AuditLogRepository
}

// NewRepositories creates all repository instances.
func NewRepositories(db *sql.DB) *Repositories {
	return &Repositories{
		Users:       NewUserRepository(db),
		Files:       NewFileRepository(db),
		FileAppeals: NewFileAppealRepository(db),
		Videos:      NewVideoRepository(db),
		Shares:      NewShareRepository(db),
		UploadTasks: NewUploadTaskRepository(db),
		AuditLogs:   NewAuditLogRepository(db),
	}
}
