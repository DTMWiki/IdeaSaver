package handler

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/gin-gonic/gin"
)

func mountFrontendStatic(r *gin.Engine, distDir string) {
	if distDir == "" {
		return
	}
	if assets := filepath.Join(distDir, "assets"); dirExists(assets) {
		r.Static("/assets", assets)
	}
	if vendor := filepath.Join(distDir, "vendor"); dirExists(vendor) {
		r.Static("/vendor", vendor)
	}
	if favicon := filepath.Join(distDir, "favicon.svg"); fileExists(favicon) {
		r.StaticFile("/favicon.svg", favicon)
	}
	if icons := filepath.Join(distDir, "icons.svg"); fileExists(icons) {
		r.StaticFile("/icons.svg", icons)
	}
}

func resolveFrontendDistDir() string {
	indexPath := resolveFrontendIndexPath()
	return filepath.Dir(indexPath)
}

func resolveFrontendIndexPath() string {
	candidates := []string{
		filepath.Join(".", "web", "dist", "index.html"),
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append([]string{
			filepath.Join(exeDir, "web", "dist", "index.html"),
			filepath.Join(exeDir, "..", "web", "dist", "index.html"),
		}, candidates...)
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}

	return filepath.Join(".", "web", "dist", "index.html")
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// cookieSecure is always true so session/auth cookies never travel over cleartext HTTP.
// Local development should use HTTPS or localhost (treated as secure by modern browsers).
func cookieSecure(_ *config.Config) bool {
	return true
}

func newOAuthState() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// --- Auth Handlers ---
