package ratelimit

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestRetryAfterDefersFutureRequests(t *testing.T) {
	t.Parallel()

	limiter := New(180)
	headers := http.Header{
		"Ratelimit-Reset":     {"1"},
		"Ratelimit-Remaining": {"1"},
	}

	delay := limiter.RetryAfter(headers, time.Now(), time.Second)
	if delay <= 0 {
		t.Fatalf("RetryAfter delay = %s, want > 0", delay)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := limiter.Wait(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Wait error = %v, want context deadline exceeded", err)
	}
}
