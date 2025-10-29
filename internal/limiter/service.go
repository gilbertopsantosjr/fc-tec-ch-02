package limiter

import (
	"context"
	"time"

	"fc-tec-ch-02/internal/config"
	"fc-tec-ch-02/internal/storage"
)

// Service manages rate limiters for different criteria (IP, Token, etc.)
type Service struct {
	ipLimiter    *RateLimiter
	tokenLimiter *RateLimiter
	storage      storage.Storage
	config       *config.Config
}

// NewService creates a new rate limiter service
func NewService(storage storage.Storage, cfg *config.Config) *Service {
	ipLimiter := NewRateLimiter(storage, cfg.MaxRequestsPerSecond, cfg.BlockingTime)
	
	return &Service{
		ipLimiter:    ipLimiter,
		tokenLimiter: ipLimiter, // Default to same limiter for tokens
		storage:      storage,
		config:       cfg,
	}
}

// CheckIP checks if a request is allowed for the given IP address
func (s *Service) CheckIP(ctx context.Context, ip string) (bool, time.Time, error) {
	if !s.config.EnableIPRateLimiter {
		return true, time.Time{}, nil
	}
	return s.ipLimiter.Check(ctx, "ip:"+ip)
}

// IncrementIP increments the request count for the given IP address
func (s *Service) IncrementIP(ctx context.Context, ip string) (int, time.Time, error) {
	if !s.config.EnableIPRateLimiter {
		return 0, time.Time{}, nil
	}
	return s.ipLimiter.Increment(ctx, "ip:"+ip)
}

// CheckToken checks if a request is allowed for the given token
// Returns the specific limits for that token if configured
func (s *Service) CheckToken(ctx context.Context, token string) (bool, time.Time, error) {
	if !s.config.EnableTokenRateLimiter {
		return true, time.Time{}, nil
	}

	// Check if token has specific limits configured
	if tokenLimit, exists := s.config.TokenLimits[token]; exists {
		// Create a temporary limiter with token-specific limits
		tokenLimiter := NewRateLimiter(s.storage, tokenLimit.MaxRequests, tokenLimit.TTL)
		return tokenLimiter.Check(ctx, "token:"+token)
	}

	// Use default IP limiter limits for unconfigured tokens
	return s.ipLimiter.Check(ctx, "token:"+token)
}

// IncrementToken increments the request count for the given token
func (s *Service) IncrementToken(ctx context.Context, token string) (int, time.Time, error) {
	if !s.config.EnableTokenRateLimiter {
		return 0, time.Time{}, nil
	}

	// Check if token has specific limits configured
	if tokenLimit, exists := s.config.TokenLimits[token]; exists {
		// Create a temporary limiter with token-specific limits
		tokenLimiter := NewRateLimiter(s.storage, tokenLimit.MaxRequests, tokenLimit.TTL)
		return tokenLimiter.Increment(ctx, "token:"+token)
	}

	// Use default limiter for unconfigured tokens
	return s.ipLimiter.Increment(ctx, "token:"+token)
}

// CheckAndIncrement checks both IP and Token, and increments the appropriate counter
// Token limits override IP limits when a token is provided
func (s *Service) CheckAndIncrement(ctx context.Context, ip, token string) (bool, time.Time, error) {
	// If token is provided, check token first (token limits override IP limits)
	if token != "" {
		allowed, resetTime, err := s.CheckToken(ctx, token)
		if !allowed {
			return false, resetTime, err
		}

		// Increment token counter
		_, _, _ = s.IncrementToken(ctx, token)
		return true, resetTime, nil
	}

	// No token provided, check IP
	allowed, resetTime, err := s.CheckIP(ctx, ip)
	if !allowed {
		return false, resetTime, err
	}

	// Increment IP counter
	_, _, _ = s.IncrementIP(ctx, ip)
	return true, resetTime, nil
}

