package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/DTMWiki/IdeaSaver/server/internal/handler"
	"github.com/DTMWiki/IdeaSaver/server/internal/middleware"
	"github.com/DTMWiki/IdeaSaver/server/internal/repository"
	"github.com/DTMWiki/IdeaSaver/server/internal/service"
	"github.com/DTMWiki/IdeaSaver/server/internal/storage"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := repository.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// One-shot CLI modes (no HTTP server)
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "migrate":
			if err := repository.Migrate(db); err != nil {
				log.Fatalf("Migration failed: %v", err)
			}
			log.Println("Migration completed successfully")
			return
		case "migrate-status":
			applied, pending, err := repository.ListMigrationStatus(db)
			if err != nil {
				log.Fatalf("Migration status failed: %v", err)
			}
			log.Printf("Applied (%d): %v", len(applied), applied)
			log.Printf("Pending (%d): %v", len(pending), pending)
			return
		case "recalc-storage":
			repos := repository.NewRepositories(db)
			n, err := repos.Users.RecalcAllStorageUsed(context.Background())
			if err != nil {
				log.Fatalf("Recalc storage failed: %v", err)
			}
			log.Printf("Recalculated storage_used for %d user(s)", n)
			return
		}
	}

	// Initialize storage clients
	ossClient, err := storage.NewOSSClient(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize OSS client: %v", err)
	}

	vcloudClient := storage.NewVCloudClient(cfg)

	// Initialize repositories
	repos := repository.NewRepositories(db)

	// Initialize services
	services := service.NewServices(cfg, repos, ossClient, vcloudClient)

	// Setup Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Never trust all proxies by default (ClientIP / rate-limit spoofing).
	// Set IDEASAVER_TRUSTED_PROXIES to your reverse-proxy CIDRs when behind nginx/CDN.
	if len(cfg.TrustedProxies) == 0 {
		_ = r.SetTrustedProxies(nil)
	} else if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Fatalf("Invalid IDEASAVER_TRUSTED_PROXIES: %v", err)
	}

	// Global middleware
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg))
	r.Use(middleware.RateLimit(cfg))
	r.Use(middleware.RequestMeta())

	// Setup routes
	handler.SetupRoutes(r, cfg, services)

	// Background trash retention (matches UI "30 days auto cleanup").
	go runTrashCleanupLoop(services.File, cfg.TrashRetentionDays)

	// Start server
	addr := ":" + cfg.Port
	log.Printf("IdeaSaver server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func runTrashCleanupLoop(files *service.FileService, retentionDays int) {
	if files == nil {
		return
	}
	// First run shortly after boot, then daily.
	timer := time.NewTimer(2 * time.Minute)
	defer timer.Stop()
	for {
		<-timer.C
		n, err := files.CleanupTrash(context.Background())
		if err != nil {
			log.Printf("trash cleanup error: %v", err)
		} else if n > 0 {
			log.Printf("trash cleanup removed %d expired item(s) (retention=%dd)", n, retentionDays)
		}
		timer.Reset(24 * time.Hour)
	}
}
