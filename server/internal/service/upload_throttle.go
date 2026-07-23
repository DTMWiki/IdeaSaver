package service

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

// uploadGate limits concurrent in-flight chunk uploads process-wide.
type uploadGate struct {
	sem chan struct{}
}

func newUploadGate(maxConcurrent int) *uploadGate {
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	return &uploadGate{sem: make(chan struct{}, maxConcurrent)}
}

func (g *uploadGate) acquire(ctx context.Context) error {
	if g == nil || g.sem == nil {
		return nil
	}
	select {
	case g.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(30 * time.Second):
		return fmt.Errorf("当前上传并发已满，请稍后重试")
	}
}

func (g *uploadGate) release() {
	if g == nil || g.sem == nil {
		return
	}
	select {
	case <-g.sem:
	default:
	}
}

// rateLimitedReader throttles reads to approximately bytesPerSec.
type rateLimitedReader struct {
	r            io.Reader
	bytesPerSec  float64
	mu           sync.Mutex
	windowStart  time.Time
	windowBytes  int64
}

func newRateLimitedReader(r io.Reader, mbps float64) io.Reader {
	if mbps <= 0 || r == nil {
		return r
	}
	return &rateLimitedReader{
		r:           r,
		bytesPerSec: mbps * 1024 * 1024,
		windowStart: time.Now(),
	}
}

func (r *rateLimitedReader) Read(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Cap single read so we can throttle progressively.
	maxChunk := int(r.bytesPerSec / 10) // ~100ms of data
	if maxChunk < 8*1024 {
		maxChunk = 8 * 1024
	}
	if maxChunk > len(p) {
		maxChunk = len(p)
	}
	if maxChunk <= 0 {
		maxChunk = len(p)
	}

	n, err := r.r.Read(p[:maxChunk])
	if n <= 0 {
		return n, err
	}

	r.windowBytes += int64(n)
	elapsed := time.Since(r.windowStart).Seconds()
	if elapsed <= 0 {
		elapsed = 0.001
	}
	allowed := r.bytesPerSec * elapsed
	if float64(r.windowBytes) > allowed {
		over := float64(r.windowBytes) - allowed
		sleep := time.Duration(over/r.bytesPerSec*1000) * time.Millisecond
		if sleep > 0 {
			time.Sleep(sleep)
		}
		// Reset window after sleep to avoid unbounded accumulation.
		r.windowStart = time.Now()
		r.windowBytes = 0
	} else if elapsed > 1 {
		r.windowStart = time.Now()
		r.windowBytes = 0
	}
	return n, err
}
