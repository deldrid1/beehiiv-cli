package ratelimit

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Limiter struct {
	mu       sync.Mutex
	interval time.Duration
	next     time.Time
}

func New(requestsPerMinute int) *Limiter {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 150
	}
	return &Limiter{
		interval: time.Minute / time.Duration(requestsPerMinute),
	}
}

func (l *Limiter) Wait(ctx context.Context) error {
	l.mu.Lock()
	now := time.Now()
	start := now
	if l.next.After(start) {
		start = l.next
	}
	l.next = start.Add(l.interval)
	delay := start.Sub(now)
	l.mu.Unlock()

	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (l *Limiter) Observe(headers http.Header, now time.Time) {
	remaining, okRemaining := headerInt(headers, "ratelimit-remaining")
	resetAt, okReset := ResetTime(headers, now)
	if !okRemaining || !okReset || remaining > 1 {
		return
	}
	l.deferUntil(resetAt)
}

func (l *Limiter) RetryAfter(headers http.Header, now time.Time, fallback time.Duration) time.Duration {
	resetAt, ok := ResetTime(headers, now)
	if !ok {
		return fallback
	}
	delay := time.Until(resetAt)
	if delay < 0 {
		return 0
	}
	l.deferUntil(resetAt)
	return delay
}

func ResetTime(headers http.Header, now time.Time) (time.Time, bool) {
	reset, ok := headerInt(headers, "ratelimit-reset")
	if !ok {
		return time.Time{}, false
	}
	if reset < 1_000_000_000 {
		return now.Add(time.Duration(reset) * time.Second), true
	}
	return time.Unix(int64(reset), 0), true
}

func headerInt(headers http.Header, key string) (int, bool) {
	for headerKey, values := range headers {
		if !strings.EqualFold(headerKey, key) || len(values) == 0 {
			continue
		}
		value, err := strconv.Atoi(values[0])
		if err != nil {
			return 0, false
		}
		return value, true
	}
	return 0, false
}

func (l *Limiter) deferUntil(next time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if next.After(l.next) {
		l.next = next
	}
}
