package limiter

import (
	"context"
	"testing"
	"time"

	"fc-tec-ch-02/internal/config"
	"fc-tec-ch-02/internal/storage"
)

// mockStorage is a mock implementation of storage.Storage for testing
type mockStorage struct {
	data          map[string]*storage.RateLimitInfo
	incrementCalls map[string]int
	getCalls       map[string]int
	clearCalls     map[string]int
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		data:          make(map[string]*storage.RateLimitInfo),
		incrementCalls: make(map[string]int),
		getCalls:       make(map[string]int),
		clearCalls:     make(map[string]int),
	}
}

func (m *mockStorage) Increment(ctx context.Context, key string, ttl time.Duration) (int, time.Time, error) {
	m.incrementCalls[key]++
	
	if info, exists := m.data[key]; exists {
		if time.Now().After(info.ResetTime) {
			// Reset if expired
			m.data[key] = &storage.RateLimitInfo{
				Count:     1,
				ResetTime: time.Now().Add(ttl),
			}
			return 1, m.data[key].ResetTime, nil
		}
		info.Count++
		return info.Count, info.ResetTime, nil
	}
	
	// First request
	resetTime := time.Now().Add(ttl)
	m.data[key] = &storage.RateLimitInfo{
		Count:     1,
		ResetTime: resetTime,
	}
	return 1, resetTime, nil
}

func (m *mockStorage) Get(ctx context.Context, key string) (*storage.RateLimitInfo, error) {
	m.getCalls[key]++
	
	if info, exists := m.data[key]; exists {
		// Return the info even if expired - let Check decide what to do
		// Return a copy to avoid mutation
		return &storage.RateLimitInfo{
			Count:     info.Count,
			ResetTime: info.ResetTime,
		}, nil
	}
	return nil, nil
}

func (m *mockStorage) Set(ctx context.Context, key string, count int, ttl time.Duration) error {
	m.data[key] = &storage.RateLimitInfo{
		Count:     count,
		ResetTime: time.Now().Add(ttl),
	}
	return nil
}

func (m *mockStorage) Clear(ctx context.Context, key string) error {
	m.clearCalls[key]++
	delete(m.data, key)
	return nil
}

func (m *mockStorage) Ping(ctx context.Context) error {
	return nil
}

func (m *mockStorage) Close() error {
	return nil
}

func TestService_CheckAndIncrement_IPOnly(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockStorage()
	
	cfg := &config.Config{
		MaxRequestsPerSecond:    5,
		BlockingTime:            1 * time.Minute,
		EnableIPRateLimiter:     true,
		EnableTokenRateLimiter:  false,
		TokenLimits:             make(map[string]config.TokenLimit),
	}
	
	service := NewService(mockStore, cfg)
	
	// Test: First 5 requests should be allowed
	for i := 0; i < 5; i++ {
		allowed, _, err := service.CheckAndIncrement(ctx, "192.168.1.1", "")
		if err != nil {
			t.Fatalf("Unexpected error on request %d: %v", i+1, err)
		}
		if !allowed {
			t.Errorf("Request %d should be allowed, but wasn't", i+1)
		}
	}
	
	// Test: 6th request should be blocked
	allowed, resetTime, _ := service.CheckAndIncrement(ctx, "192.168.1.1", "")
	// Error is allowed when limit is exceeded (ErrLimitExceeded)
	if allowed {
		t.Error("6th request should be blocked, but wasn't")
	}
	if resetTime.IsZero() {
		t.Error("Reset time should not be zero")
	}
	
	// Verify storage was called
	if mockStore.getCalls["ip:192.168.1.1"] < 5 {
		t.Errorf("Expected at least 5 Get calls, got %d", mockStore.getCalls["ip:192.168.1.1"])
	}
}

func TestService_CheckAndIncrement_WithToken(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockStorage()
	
	cfg := &config.Config{
		MaxRequestsPerSecond:    5,
		BlockingTime:            1 * time.Minute,
		EnableIPRateLimiter:     true,
		EnableTokenRateLimiter:  true,
		TokenLimits:             make(map[string]config.TokenLimit),
	}
	
	service := NewService(mockStore, cfg)
	
	// Test: Token should override IP rate limiting
	token := "test-token-123"
	
	// Make 5 requests with token
	for i := 0; i < 5; i++ {
		allowed, _, err := service.CheckAndIncrement(ctx, "192.168.1.1", token)
		if err != nil {
			t.Fatalf("Unexpected error on request %d: %v", i+1, err)
		}
		if !allowed {
			t.Errorf("Request %d with token should be allowed, but wasn't", i+1)
		}
	}
	
	// 6th request with token should be blocked
	allowed, _, err := service.CheckAndIncrement(ctx, "192.168.1.1", token)
	// Error is allowed when limit is exceeded
	if allowed {
		t.Error("6th request with token should be blocked, but wasn't")
	}
	
	// IP-based requests should still work (separate counter)
	allowed, _, err = service.CheckAndIncrement(ctx, "192.168.1.1", "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !allowed {
		t.Error("IP-based request should be allowed (separate from token counter)")
	}
	
	// Verify token storage was used
	if mockStore.getCalls["token:test-token-123"] == 0 {
		t.Error("Expected token storage to be used")
	}
}

func TestService_CheckAndIncrement_TokenWithSpecificLimits(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockStorage()
	
	cfg := &config.Config{
		MaxRequestsPerSecond:    5,
		BlockingTime:            1 * time.Minute,
		EnableIPRateLimiter:     true,
		EnableTokenRateLimiter:  true,
		TokenLimits: map[string]config.TokenLimit{
			"premium-token": {
				MaxRequests: 10,
				TTL:         2 * time.Minute,
			},
		},
	}
	
	service := NewService(mockStore, cfg)
	
	// Test: Premium token should have higher limit (10 requests)
	token := "premium-token"
	
	// Make 10 requests with premium token (should all be allowed)
	for i := 0; i < 10; i++ {
		allowed, _, err := service.CheckAndIncrement(ctx, "192.168.1.1", token)
		if err != nil {
			t.Fatalf("Unexpected error on request %d: %v", i+1, err)
		}
		if !allowed {
			t.Errorf("Premium token request %d should be allowed, but wasn't", i+1)
		}
	}
	
	// 11th request should be blocked
	allowed, _, _ := service.CheckAndIncrement(ctx, "192.168.1.1", token)
	// Error is allowed when limit is exceeded
	if allowed {
		t.Error("11th request with premium token should be blocked, but wasn't")
	}
}

func TestService_CheckAndIncrement_IPRateLimiterDisabled(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockStorage()
	
	cfg := &config.Config{
		MaxRequestsPerSecond:    5,
		BlockingTime:            1 * time.Minute,
		EnableIPRateLimiter:     false,
		EnableTokenRateLimiter:  false,
		TokenLimits:             make(map[string]config.TokenLimit),
	}
	
	service := NewService(mockStore, cfg)
	
	// Test: All requests should be allowed when rate limiter is disabled
	for i := 0; i < 20; i++ {
		allowed, _, err := service.CheckAndIncrement(ctx, "192.168.1.1", "")
		if err != nil {
			t.Fatalf("Unexpected error on request %d: %v", i+1, err)
		}
		if !allowed {
			t.Errorf("Request %d should be allowed when rate limiter is disabled, but wasn't", i+1)
		}
	}
	
	// Verify storage was not called
	if len(mockStore.getCalls) > 0 {
		t.Error("Storage should not be called when rate limiter is disabled")
	}
}

func TestService_CheckAndIncrement_DifferentIPs(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockStorage()
	
	cfg := &config.Config{
		MaxRequestsPerSecond:    3,
		BlockingTime:            1 * time.Minute,
		EnableIPRateLimiter:     true,
		EnableTokenRateLimiter:  false,
		TokenLimits:             make(map[string]config.TokenLimit),
	}
	
	service := NewService(mockStore, cfg)
	
	// Test: Different IPs should have separate rate limit counters
	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"
	
	// Exhaust IP1's limit
	for i := 0; i < 3; i++ {
		allowed, _, err := service.CheckAndIncrement(ctx, ip1, "")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !allowed {
			t.Errorf("IP1 request %d should be allowed", i+1)
		}
	}
	
	// IP1 should now be blocked
	allowed, _, err := service.CheckAndIncrement(ctx, ip1, "")
	// Error is allowed when limit is exceeded
	if allowed {
		t.Error("IP1 should be blocked after 3 requests")
	}
	
	// IP2 should still be allowed (separate counter)
	allowed, _, err = service.CheckAndIncrement(ctx, ip2, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !allowed {
		t.Error("IP2 should be allowed (separate counter from IP1)")
	}
}

func TestService_CheckAndIncrement_TokenOverridesIP(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockStorage()
	
	cfg := &config.Config{
		MaxRequestsPerSecond:    3,
		BlockingTime:            1 * time.Minute,
		EnableIPRateLimiter:     true,
		EnableTokenRateLimiter:  true,
		TokenLimits:             make(map[string]config.TokenLimit),
	}
	
	service := NewService(mockStore, cfg)
	ip := "192.168.1.1"
	token := "my-token"
	
	// Exhaust IP limit
	for i := 0; i < 3; i++ {
		allowed, _, err := service.CheckAndIncrement(ctx, ip, "")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !allowed {
			t.Errorf("IP request %d should be allowed", i+1)
		}
	}
	
	// IP should be blocked
	allowed, _, err := service.CheckAndIncrement(ctx, ip, "")
	// Error is allowed when limit is exceeded
	if allowed {
		t.Error("IP should be blocked after exhausting limit")
	}
	
	// Same IP with token should still be allowed (token takes precedence)
	allowed, _, err = service.CheckAndIncrement(ctx, ip, token)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !allowed {
		t.Error("Request with token should be allowed even if IP is blocked")
	}
}

