package limiter

import (
	"context"
	"errors"
	"time"

	"fc-tec-ch-02/internal/storage"
)

var (
	ErrLimitExceeded = errors.New("rate limit exceeded")
)

// RateLimiter handles rate limiting logic
type RateLimiter struct {
	storage   storage.Storage
	maxReqs   int
	blockTime time.Duration
}

// NewRateLimiter creates a new rate limiter instance
func NewRateLimiter(storage storage.Storage, maxRequests int, blockTime time.Duration) *RateLimiter {
	return &RateLimiter{
		storage:   storage,
		maxReqs:   maxRequests,
		blockTime: blockTime,
	}
}

// Check checks if a request is allowed for the given identifier
// Returns: (allowed bool, resetTime time.Time, err error)
func (rl *RateLimiter) Check(ctx context.Context, identifier string) (bool, time.Time, error) {
	// Get current rate limit info
	info, err := rl.storage.Get(ctx, identifier)
	if err != nil {
		return false, time.Time{}, err
	}

	// If no info exists, first request is allowed
	if info == nil {
		return true, time.Now().Add(rl.blockTime), nil
	}

	// Check if blocked period has expired
	if time.Now().After(info.ResetTime) {
		// Reset the count
		if err := rl.storage.Clear(ctx, identifier); err != nil {
			return false, time.Time{}, err
		}
		return true, time.Now().Add(rl.blockTime), nil
	}

	// Check if limit is exceeded
	if info.Count >= rl.maxReqs {
		return false, info.ResetTime, ErrLimitExceeded
	}

	// Allowed
	return true, info.ResetTime, nil
}

// Increment increments the request count for the given identifier
func (rl *RateLimiter) Increment(ctx context.Context, identifier string) (int, time.Time, error) {
	return rl.storage.Increment(ctx, identifier, rl.blockTime)
}


