package storage

import (
	"context"
	"time"
)

// RateLimitInfo represents the information about rate limiting for a key
type RateLimitInfo struct {
	Count     int
	ResetTime time.Time
}

// Storage defines the interface for rate limiter storage
type Storage interface {
	// Increment increments the request count for a given key
	// Returns the current count and expiration time
	Increment(ctx context.Context, key string, ttl time.Duration) (int, time.Time, error)

	// Get retrieves the current rate limit info for a given key
	Get(ctx context.Context, key string) (*RateLimitInfo, error)

	// Set explicitly sets the count and TTL for a key
	Set(ctx context.Context, key string, count int, ttl time.Duration) error

	// Clear removes a key from storage
	Clear(ctx context.Context, key string) error

	// Ping checks if the storage is available
	Ping(ctx context.Context) error

	// Close closes the storage connection
	Close() error
}


