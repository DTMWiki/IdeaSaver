package service

import (
	"context"
	"testing"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/google/uuid"
)

func TestInitUploadRejectsNonPositiveSize(t *testing.T) {
	svc := NewUploadService(&config.Config{MaxUploadSizeMB: 100}, nil, nil, nil)
	_, err := svc.InitUpload(context.Background(), uuid.New(), &InitUploadRequest{
		Filename: "a.bin",
		Size:     0,
	})
	if err == nil {
		t.Fatal("expected error for size=0")
	}
}
