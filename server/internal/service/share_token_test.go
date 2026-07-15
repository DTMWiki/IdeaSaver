package service

import (
	"testing"
	"time"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

func TestShareDownloadTokenIssueAndValidate(t *testing.T) {
	svc := NewShareService(&config.Config{JWTSecret: "unit-test-secret"}, nil, nil)

	token, expiresIn, err := svc.issueDownloadToken("abc12345")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if expiresIn != int(shareDownloadTokenTTL.Seconds()) {
		t.Fatalf("expiresIn=%d", expiresIn)
	}
	if err := svc.validateDownloadToken("abc12345", token); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if err := svc.validateDownloadToken("other-code", token); err != ErrShareTokenInvalid {
		t.Fatalf("expected code mismatch invalid, got %v", err)
	}
	if err := svc.validateDownloadToken("abc12345", "not.a.jwt"); err != ErrShareTokenInvalid {
		t.Fatalf("expected garbage invalid, got %v", err)
	}
}

func TestShareDownloadTokenRejectsWrongType(t *testing.T) {
	svc := NewShareService(&config.Config{JWTSecret: "unit-test-secret"}, nil, nil)
	claims := jwt.MapClaims{
		"typ":  "session",
		"code": "abc12345",
		"exp":  time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte("unit-test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.validateDownloadToken("abc12345", signed); err != ErrShareTokenInvalid {
		t.Fatalf("expected typ mismatch, got %v", err)
	}
}

func TestShareDownloadTokenRejectsExpired(t *testing.T) {
	svc := NewShareService(&config.Config{JWTSecret: "unit-test-secret"}, nil, nil)
	claims := jwt.MapClaims{
		"typ":  "share_dl",
		"code": "abc12345",
		"exp":  time.Now().Add(-time.Minute).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte("unit-test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.validateDownloadToken("abc12345", signed); err != ErrShareTokenInvalid {
		t.Fatalf("expected expired invalid, got %v", err)
	}
}
