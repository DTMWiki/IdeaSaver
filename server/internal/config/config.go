package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application.
type Config struct {
	// Server
	Environment string
	Port        string

	// Database
	DatabaseURL string

	// Authelia OAuth2
	AutheliaIssuer       string
	AutheliaClientID     string
	AutheliaClientSecret string
	AutheliaRedirectURL  string
	OIDCAuthURL          string
	OIDCTokenURL         string
	OIDCUserInfoURL      string
	OIDCScopes           string

	// JWT
	JWTSecret string

	// DogeCloud OSS
	DogeAccessKey string
	DogeSecretKey string
	DogeBucket    string
	DogeEndpoint  string
	DogeRegion    string
	PublicBaseURL string // e.g. https://i.dtmwiki.cn

	// DogeCloud VCloud
	DogeVCloudAPI string
	DogeUserID    string

	// Upload limits
	MaxUploadSizeMB      int64
	MaxConcurrentUploads int
	UploadRateLimitMBps  float64 // MB/s, 0 = unlimited

	// User defaults
	DefaultQuotaBytes int64 // default storage quota per user

	// Trash cleanup
	TrashRetentionDays int
}

// Load reads configuration from environment variables and .env file.
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	cfg := &Config{
		Environment: getEnv("IDEASAVER_ENV", "development"),
		Port:        getEnv("IDEASAVER_PORT", "8080"),

		DatabaseURL: getEnv("IDEASAVER_DATABASE_URL", ""),

		AutheliaIssuer:       getEnv("IDEASAVER_AUTHELIA_ISSUER", ""),
		AutheliaClientID:     getEnv("IDEASAVER_AUTHELIA_CLIENT_ID", ""),
		AutheliaClientSecret: getEnv("IDEASAVER_AUTHELIA_CLIENT_SECRET", ""),
		AutheliaRedirectURL:  getEnv("IDEASAVER_AUTHELIA_REDIRECT_URL", ""),
		OIDCAuthURL:          getEnv("IDEASAVER_OIDC_AUTH_URL", ""),
		OIDCTokenURL:         getEnv("IDEASAVER_OIDC_TOKEN_URL", ""),
		OIDCUserInfoURL:      getEnv("IDEASAVER_OIDC_USERINFO_URL", ""),
		OIDCScopes:           getEnv("IDEASAVER_OIDC_SCOPES", ""),

		JWTSecret: getEnv("IDEASAVER_JWT_SECRET", ""),

		DogeAccessKey: getEnv("IDEASAVER_DOGE_ACCESS_KEY", ""),
		DogeSecretKey: getEnv("IDEASAVER_DOGE_SECRET_KEY", ""),
		DogeBucket:    getEnv("IDEASAVER_DOGE_BUCKET", ""),
		DogeEndpoint:  getEnv("IDEASAVER_DOGE_ENDPOINT", ""),
		DogeRegion:    getEnv("IDEASAVER_DOGE_REGION", ""),
		PublicBaseURL: getEnv("IDEASAVER_PUBLIC_BASE_URL", "https://i.dtmwiki.cn"),

		DogeVCloudAPI: getEnv("IDEASAVER_DOGE_VCLOUD_API", "https://api.dogecloud.com"),
		DogeUserID:    getEnv("IDEASAVER_DOGE_USER_ID", ""),

		MaxUploadSizeMB:      getEnvInt64("IDEASAVER_MAX_UPLOAD_SIZE_MB", 10240), // 10GB
		MaxConcurrentUploads: int(getEnvInt64("IDEASAVER_MAX_CONCURRENT_UPLOADS", 10)),
		UploadRateLimitMBps:  getEnvFloat("IDEASAVER_UPLOAD_RATE_LIMIT_MBPS", 0),       // 0 = unlimited
		DefaultQuotaBytes:    getEnvInt64("IDEASAVER_DEFAULT_QUOTA_BYTES", 5368709120), // 5GB
		TrashRetentionDays:   int(getEnvInt64("IDEASAVER_TRASH_RETENTION_DAYS", 30)),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("IDEASAVER_DATABASE_URL is required")
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("IDEASAVER_JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt64(key string, defaultVal int64) int64 {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return defaultVal
}
