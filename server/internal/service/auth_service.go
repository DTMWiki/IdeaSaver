package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	cfg      *config.Config
	userRepo *repository.UserRepository
	oauth2   *oauth2.Config
}

func NewAuthService(cfg *config.Config, userRepo *repository.UserRepository) *AuthService {
	return &AuthService{
		cfg:      cfg,
		userRepo: userRepo,
		oauth2: &oauth2.Config{
			ClientID:     cfg.AutheliaClientID,
			ClientSecret: cfg.AutheliaClientSecret,
			RedirectURL:  cfg.AutheliaRedirectURL,
			Scopes:       []string{"openid", "profile", "email", "groups"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  cfg.AutheliaIssuer + "/api/oidc/authorization",
				TokenURL: cfg.AutheliaIssuer + "/api/oidc/token",
			},
		},
	}
}

// GetAuthURL returns the OAuth2 authorization URL.
func (s *AuthService) GetAuthURL(state string) string {
	return s.oauth2.AuthCodeURL(state)
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
	resp, err := client.Get(s.cfg.AutheliaIssuer + "/api/oidc/userinfo")
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
