package utils

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

const (
	maxRetries   = 5
	initialDelay = 2 * time.Second
	maxDelay     = 60 * time.Second
)

// RateLimitError is a provider error carrying an HTTP 429 (status, and
// the Retry-After delay when the engine sent one).
type RateLimitError interface {
	error
	Status() int
	RetryAfter() *time.Duration
}

// IsRateLimitError mirrors isRateLimitError.
func IsRateLimitError(err error) bool {
	var rl RateLimitError
	return errors.As(err, &rl) && rl.Status() == 429
}

// RateLimitDelay mirrors rateLimitDelay.
func RateLimitDelay(attempt int, retryAfter *time.Duration, random func() float64) time.Duration {
	if retryAfter != nil {
		if *retryAfter < maxDelay {
			return *retryAfter
		}
		return maxDelay
	}
	exponential := time.Duration(math.Min(float64(initialDelay)*math.Pow(2, float64(attempt)), float64(maxDelay)))
	ms := math.Round(float64(exponential.Milliseconds()) * (0.5 + random()*0.5))
	return time.Duration(ms) * time.Millisecond
}

// RetryOnRateLimit mirrors retryOnRateLimit: rate limited provider calls
// are retried with backoff.
func RetryOnRateLimit(ctx context.Context, label string, logger Logger, operation func() error) error {
	attempt := 0
	for {
		err := operation()
		if err == nil {
			return nil
		}
		if !IsRateLimitError(err) || attempt >= maxRetries {
			return err
		}
		var rl RateLimitError
		errors.As(err, &rl)
		delay := RateLimitDelay(attempt, rl.RetryAfter(), rand.Float64)
		if logger != nil {
			logger.Warn(fmt.Sprintf("[Search] Rate limited on %s: waiting %dms before retry %d/%d (%s)", label, delay.Milliseconds(), attempt+1, maxRetries, err.Error()))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
		attempt++
	}
}
