package service

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

func TestUploadGateLimitsConcurrency(t *testing.T) {
	g := newUploadGate(1)
	if err := g.acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := g.acquire(ctx); err == nil {
		t.Fatal("expected second acquire to fail while first held")
	}
	g.release()
	if err := g.acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
	g.release()
}

func TestRateLimitedReaderPassesData(t *testing.T) {
	src := bytes.Repeat([]byte("a"), 32*1024)
	r := newRateLimitedReader(bytes.NewReader(src), 100) // 100 MB/s — effectively fast
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, src) {
		t.Fatalf("data mismatch len=%d", len(out))
	}
}

func TestRateLimitedReaderUnlimitedPassthrough(t *testing.T) {
	src := []byte("hello")
	r := newRateLimitedReader(bytes.NewReader(src), 0)
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "hello" {
		t.Fatalf("got %q", out)
	}
}
