package limiter

import (
	"context"
	"testing"
	"time"

	"fc-tec-ch-02/internal/storage"
)

func TestRateLimiter_Check_FirstRequest(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockStorage()
	
	rl := NewRateLimiter(mockStore, 5, 1*time.Minute)
	
	// First request should be allowed
	allowed, resetTime, err := rl.Check(ctx, "test-key")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !allowed {
		t.Error("First request should be allowed")
	}
	if resetTime.IsZero() {
		t.Error("Reset time should not be zero")
	}
	
	// Verify storage was queried
	if mockStore.getCalls["test-key"] != 1 {
		t.Errorf("Expected 1 Get call, got %d", mockStore.getCalls["test-key"])
	}
}

func TestRateLimiter_Check_WithinLimit(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockStorage()
	
	rl := NewRateLimiter(mockStore, 3, 1*time.Minute)
	
	// Set initial count to 2 (below limit)
	mockStore.Set(ctx, "test-key", 2, 1*time.Minute)
	
	// Request should be allowed
	allowed, _, err := rl.Check(ctx, "test-key")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !allowed {
		t.Error("Request should be allowed when count is below limit")
	}
}

func TestRateLimiter_Check_AtLimit(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockStorage()
	
	rl := NewRateLimiter(mockStore, 3, 1*time.Minute)
	
	// Set count to exactly the limit
	resetTime := time.Now().Add(1 * time.Minute)
	mockStore.data["test-key"] = &storage.RateLimitInfo{
		Count:     3,
		ResetTime: resetTime,
	}
	
	// Request should be blocked
	allowed, returnedResetTime, err := rl.Check(ctx, "test-key")
	if err == nil || err != ErrLimitExceeded {
		t.Errorf("Expected ErrLimitExceeded, got: %v", err)
	}
	if allowed {
		t.Error("Request should be blocked when count equals limit")
	}
	if !returnedResetTime.Equal(resetTime) {
		t.Errorf("Reset time mismatch: got %v, want %v", returnedResetTime, resetTime)
	}
}

func TestRateLimiter_Check_ExceededLimit(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockStorage()
	
	rl := NewRateLimiter(mockStore, 3, 1*time.Minute)
	
	// Set count above the limit
	resetTime := time.Now().Add(1 * time.Minute)
	mockStore.data["test-key"] = &storage.RateLimitInfo{
		Count:     5,
		ResetTime: resetTime,
	}
	
	// Request should be blocked
	allowed, _, err := rl.Check(ctx, "test-key")
	if err == nil || err != ErrLimitExceeded {
		t.Errorf("Expected ErrLimitExceeded, got: %v", err)
	}
	if allowed {
		t.Error("Request should be blocked when count exceeds limit")
	}
}

func TestRateLimiter_Check_ExpiredReset(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockStorage()
	
	rl := NewRateLimiter(mockStore, 3, 1*time.Minute)
	
	// Set count with expired reset time
	expiredResetTime := time.Now().Add(-1 * time.Minute) // In the past
	mockStore.data["test-key"] = &storage.RateLimitInfo{
		Count:     5,
		ResetTime: expiredResetTime,
	}
	
	// Request should be allowed (expired limit resets)
	allowed, resetTime, err := rl.Check(ctx, "test-key")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !allowed {
		t.Error("Request should be allowed when reset time has expired")
	}
	if resetTime.IsZero() {
		t.Error("Reset time should not be zero after expiration reset")
	}
	
	// Verify storage was cleared
	if mockStore.clearCalls["test-key"] == 0 {
		t.Error("Expected Clear to be called for expired limit")
	}
}

func TestRateLimiter_Increment_FirstRequest(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockStorage()
	
	rl := NewRateLimiter(mockStore, 5, 1*time.Minute)
	
	// First increment
	count, resetTime, err := rl.Increment(ctx, "test-key")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}
	if resetTime.IsZero() {
		t.Error("Reset time should not be zero")
	}
	
	// Verify storage was updated
	if mockStore.incrementCalls["test-key"] != 1 {
		t.Errorf("Expected 1 Increment call, got %d", mockStore.incrementCalls["test-key"])
	}
}

func TestRateLimiter_Increment_MultipleRequests(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockStorage()
	
	rl := NewRateLimiter(mockStore, 5, 1*time.Minute)
	
	// Make multiple increments
	expectedCount := 1
	for i := 0; i < 5; i++ {
		count, _, err := rl.Increment(ctx, "test-key")
		if err != nil {
			t.Fatalf("Unexpected error on increment %d: %v", i+1, err)
		}
		if count != expectedCount {
			t.Errorf("Increment %d: expected count %d, got %d", i+1, expectedCount, count)
		}
		expectedCount++
	}
	
	// Verify storage was called correct number of times
	if mockStore.incrementCalls["test-key"] != 5 {
		t.Errorf("Expected 5 Increment calls, got %d", mockStore.incrementCalls["test-key"])
	}
}

func TestRateLimiter_Increment_AfterExpiration(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockStorage()
	
	rl := NewRateLimiter(mockStore, 5, 1*time.Minute)
	
	// Set expired entry
	expiredResetTime := time.Now().Add(-1 * time.Minute)
	mockStore.data["test-key"] = &storage.RateLimitInfo{
		Count:     10,
		ResetTime: expiredResetTime,
	}
	
	// Increment should reset the counter
	count, resetTime, err := rl.Increment(ctx, "test-key")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count to reset to 1, got %d", count)
	}
	if resetTime.IsZero() {
		t.Error("Reset time should not be zero")
	}
	if time.Now().After(resetTime) {
		t.Error("Reset time should be in the future")
	}
}

func TestRateLimiter_Integration(t *testing.T) {
	ctx := context.Background()
	mockStore := newMockStorage()
	
	rl := NewRateLimiter(mockStore, 3, 1*time.Minute)
	
	// Simulate workflow: Check, then Increment
	// Request 1
	allowed, _, err := rl.Check(ctx, "test-key")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !allowed {
		t.Error("First request should be allowed")
	}
	
	count, _, err := rl.Increment(ctx, "test-key")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}
	
	// Request 2
	allowed, _, err = rl.Check(ctx, "test-key")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !allowed {
		t.Error("Second request should be allowed")
	}
	
	count, _, err = rl.Increment(ctx, "test-key")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected count 2, got %d", count)
	}
	
	// Request 3 (at limit)
	allowed, _, err = rl.Check(ctx, "test-key")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !allowed {
		t.Error("Third request should still be allowed (at limit)")
	}
	
	count, _, err = rl.Increment(ctx, "test-key")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
	
	// Request 4 (should be blocked)
	allowed, _, err = rl.Check(ctx, "test-key")
	if err == nil || err != ErrLimitExceeded {
		t.Errorf("Expected ErrLimitExceeded, got: %v", err)
	}
	if allowed {
		t.Error("Fourth request should be blocked (exceeded limit)")
	}
}
