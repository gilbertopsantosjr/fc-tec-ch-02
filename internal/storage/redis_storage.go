package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStorage implements the Storage interface using Redis
type RedisStorage struct {
	client *redis.Client
}

// NewRedisStorage creates a new Redis storage instance
func NewRedisStorage(host, port string) (*RedisStorage, error) {
	redisURL := fmt.Sprintf("%s:%s", host, port)
	
	client := redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: "", // No password
		DB:       0,  // Default DB
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStorage{client: client}, nil
}

// Increment increments the request count for a given key
func (r *RedisStorage) Increment(ctx context.Context, key string, ttl time.Duration) (int, time.Time, error) {
	pipe := r.client.Pipeline()
	
	// Increment the count
	incrCmd := pipe.Incr(ctx, key)
	
	// Set expiration if this is a new key
	pipe.Expire(ctx, key, ttl)
	
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return 0, time.Time{}, fmt.Errorf("failed to increment key: %w", err)
	}

	// Get current count
	count, err := incrCmd.Result()
	if err != nil {
		count = 1
	}

	// Try to get existing reset time from a separate key
	resetTime := time.Now().Add(ttl)
	infoKey := fmt.Sprintf("%s:info", key)
	infoStr, err := r.client.Get(ctx, infoKey).Result()
	if err == nil {
		var info RateLimitInfo
		if json.Unmarshal([]byte(infoStr), &info) == nil {
			resetTime = info.ResetTime
		}
	}

	// Update reset time info
	info := RateLimitInfo{
		Count:     int(count),
		ResetTime: resetTime,
	}
	infoData, _ := json.Marshal(info)
	r.client.Set(ctx, infoKey, string(infoData), ttl)

	return int(count), resetTime, nil
}

// Get retrieves the current rate limit info for a given key
func (r *RedisStorage) Get(ctx context.Context, key string) (*RateLimitInfo, error) {
	countStr, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	count := 0
	if countStr != "" {
		_, _ = fmt.Sscanf(countStr, "%d", &count)
	}

	// Try to get reset time info
	resetTime := time.Now()
	infoKey := fmt.Sprintf("%s:info", key)
	infoStr, err := r.client.Get(ctx, infoKey).Result()
	if err == nil {
		var info RateLimitInfo
		if json.Unmarshal([]byte(infoStr), &info) == nil {
			resetTime = info.ResetTime
		}
	}

	return &RateLimitInfo{
		Count:     count,
		ResetTime: resetTime,
	}, nil
}

// Set explicitly sets the count and TTL for a key
func (r *RedisStorage) Set(ctx context.Context, key string, count int, ttl time.Duration) error {
	// Set the count
	if err := r.client.Set(ctx, key, count, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set key: %w", err)
	}

	// Set reset time info
	resetTime := time.Now().Add(ttl)
	info := RateLimitInfo{
		Count:     count,
		ResetTime: resetTime,
	}
	infoData, _ := json.Marshal(info)
	infoKey := fmt.Sprintf("%s:info", key)
	if err := r.client.Set(ctx, infoKey, string(infoData), ttl).Err(); err != nil {
		return fmt.Errorf("failed to set info key: %w", err)
	}

	return nil
}

// Clear removes a key from storage
func (r *RedisStorage) Clear(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}
	
	infoKey := fmt.Sprintf("%s:info", key)
	r.client.Del(ctx, infoKey)
	
	return nil
}

// Ping checks if the storage is available
func (r *RedisStorage) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Close closes the storage connection
func (r *RedisStorage) Close() error {
	return r.client.Close()
}

