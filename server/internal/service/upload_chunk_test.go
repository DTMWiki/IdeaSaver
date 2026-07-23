package service

import (
	"testing"

	"github.com/DTMWiki/IdeaSaver/server/internal/model"
)

func TestExpectedChunkSize(t *testing.T) {
	task := &model.UploadTask{
		TotalSize:   12 * 1024 * 1024,
		ChunkSize:   5 * 1024 * 1024,
		TotalChunks: 3,
	}
	if got := expectedChunkSize(task, 0); got != 5*1024*1024 {
		t.Fatalf("chunk0=%d", got)
	}
	if got := expectedChunkSize(task, 1); got != 5*1024*1024 {
		t.Fatalf("chunk1=%d", got)
	}
	if got := expectedChunkSize(task, 2); got != 2*1024*1024 {
		t.Fatalf("chunk2 last=%d", got)
	}

	single := &model.UploadTask{TotalSize: 100, ChunkSize: 5 * 1024 * 1024, TotalChunks: 1}
	if got := expectedChunkSize(single, 0); got != 100 {
		t.Fatalf("single=%d", got)
	}
}
