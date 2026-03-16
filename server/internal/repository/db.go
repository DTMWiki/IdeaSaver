package repository

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"

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

// Migrate runs all SQL migration files.
func Migrate(db *sql.DB) error {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		content, err := migrationFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}

		log.Printf("Running migration: %s", entry.Name())
		if _, err := db.ExecContext(context.Background(), string(content)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", entry.Name(), err)
		}
	}

	return nil
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
