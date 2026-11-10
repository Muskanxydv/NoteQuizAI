package llm

import (
	"context"
	"errors"
	"math/rand"
	"time"
)

// isRetryable decides whether another attempt could succeed.
func isRetryable(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, ErrBlocked) || errors.Is(err, ErrDailyQuota) {
		return false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Retryable()
	}
	return true // network errors and per-request timeouts
}

func retryAfterOf(err error) time.Duration {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.RetryAfter
	}
	return 0
}

// backoff returns the wait before retry number `attempt` (1-based):
// 1s, 2s, 4s... capped at 20s, with +/-25% jitter, or the server's hint if longer.
func backoff(attempt int, hint time.Duration) time.Duration {
	const maxWait = 20 * time.Second
	d := maxWait
	if attempt <= 5 {
		d = time.Second << (attempt - 1)
		if d > maxWait {
			d = maxWait
		}
	}
	d += time.Duration(rand.Int63n(int64(d)/2)) - d/4
	if hint > d {
		d = hint
	}
	return d
}

// sleepCtx waits for d or until ctx is cancelled.
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
