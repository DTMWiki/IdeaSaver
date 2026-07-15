package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/DTMWiki/IdeaSaver/server/internal/model"
	"github.com/DTMWiki/IdeaSaver/server/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// AuthService handles OAuth2 authentication with Authelia.
type AuthService struct {
	cfg         *config.Config
	userRepo    *repository.UserRepository
	auditRepo   *repository.AuditLogRepository
	oauth2      *oauth2.Config
	userInfoURL string
}

func NewAuthService(cfg *config.Config, userRepo *repository.UserRepository, auditRepo *repository.AuditLogRepository) *AuthService {
	authURL := cfg.OIDCAuthURL
	if authURL == "" {
		authURL = cfg.AutheliaIssuer + "/api/oidc/authorization"
	}
	tokenURL := cfg.OIDCTokenURL
	if tokenURL == "" {
		tokenURL = cfg.AutheliaIssuer + "/api/oidc/token"
	}
	userInfoURL := cfg.OIDCUserInfoURL
	if userInfoURL == "" {
		userInfoURL = cfg.AutheliaIssuer + "/api/oidc/userinfo"
	}

	return &AuthService{
		cfg:         cfg,
		userRepo:    userRepo,
		auditRepo:   auditRepo,
		userInfoURL: userInfoURL,
		oauth2: &oauth2.Config{
			ClientID:     cfg.AutheliaClientID,
			ClientSecret: cfg.AutheliaClientSecret,
			RedirectURL:  cfg.AutheliaRedirectURL,
			Scopes:       parseOIDCScopes(cfg.OIDCScopes),
			Endpoint: oauth2.Endpoint{
				AuthURL:  authURL,
				TokenURL: tokenURL,
			},
		},
	}
}

// BeginAuth generates a cryptographically random OAuth state and returns the
// authorization URL. Callers must persist state (e.g. HttpOnly cookie) and
// verify it on callback.
func (s *AuthService) BeginAuth() (authURL, state string, err error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("failed to generate oauth state: %w", err)
	}
	state = hex.EncodeToString(buf)
	// state is always random here — never a constant string.
	return s.oauth2.AuthCodeURL(state), state, nil
}

// GetAuthURL always generates a fresh random OAuth state (BeginAuth).
// The state argument is ignored so callers cannot pass a constant value into AuthCodeURL.
func (s *AuthService) GetAuthURL(_ string) string {
	url, _, err := s.BeginAuth()
	if err != nil {
		return ""
	}
	return url
}

// ExchangeCode exchanges an authorization code for user info and returns a JWT.
func (s *AuthService) ExchangeCode(ctx context.Context, code string) (string, *model.User, error) {
	// Exchange code for token
	token, err := s.oauth2.Exchange(ctx, code)
	if err != nil {
		return "", nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get userinfo from Authelia
	client := s.oauth2.Client(ctx, token)
	resp, err := client.Get(s.userInfoURL)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("userinfo request failed with status %d", resp.StatusCode)
	}

	var userInfo struct {
		Sub               string   `json:"sub"`
		PreferredUsername string   `json:"preferred_username"`
		Name              string   `json:"name"`
		Email             string   `json:"email"`
		Groups            []string `json:"groups"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return "", nil, fmt.Errorf("failed to decode userinfo: %w", err)
	}

	// Determine role from groups
	role := "user"
	for _, g := range userInfo.Groups {
		if g == "admin" || g == "admins" {
			role = "admin"
			break
		}
	}

	// Upsert user into database
	user := &model.User{
		Username:     userInfo.PreferredUsername,
		DisplayName:  userInfo.Name,
		Email:        userInfo.Email,
		Role:         role,
		StorageQuota: s.cfg.DefaultQuotaBytes,
	}

	if err := s.userRepo.Upsert(ctx, user); err != nil {
		return "", nil, fmt.Errorf("failed to upsert user: %w", err)
	}

	// Re-fetch to get full user data (including existing quota if not new)
	user, err = s.userRepo.FindByUsername(ctx, userInfo.PreferredUsername)
	if err != nil {
		return "", nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	// Generate JWT
	jwtToken, err := s.generateJWT(user)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate JWT: %w", err)
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Create(ctx, &model.AuditLog{
			UserID:   user.ID,
			Action:   "login",
			Resource: "user",
			Details: map[string]any{
				"role":     user.Role,
				"username": user.Username,
			},
		})
	}

	return jwtToken, user, nil
}

func (s *AuthService) generateJWT(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID.String(),
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

// GetUser retrieves a user by ID.
func (s *AuthService) GetUser(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return s.userRepo.FindByID(ctx, id)
}

// UserRepo exposes user repository for auth middleware wiring.
func (s *AuthService) UserRepo() *repository.UserRepository {
	return s.userRepo
}

func parseOIDCScopes(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{"openid", "profile", "email", "groups"}
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
	scopes := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			scopes = append(scopes, p)
		}
	}
	if len(scopes) == 0 {
		return []string{"openid", "profile", "email", "groups"}
	}
	return scopes
}
