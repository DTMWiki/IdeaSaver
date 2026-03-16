package main

import (
	"log"
	"os"

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

	// Run migrations if requested
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := repository.Migrate(db); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		log.Println("Migration completed successfully")
		return
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

	// Global middleware
	r.Use(middleware.CORS(cfg))
	r.Use(middleware.RateLimit(cfg))

	// Setup routes
	handler.SetupRoutes(r, cfg, services)

	// Start server
	addr := ":" + cfg.Port
	log.Printf("IdeaSaver server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
