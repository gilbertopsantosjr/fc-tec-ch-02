package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort              string
	RedisHost               string
	RedisPort               string
	MaxRequestsPerSecond    int
	BlockingTime            time.Duration
	TokenLimits             map[string]TokenLimit
	EnableIPRateLimiter     bool
	EnableTokenRateLimiter  bool
}

type TokenLimit struct {
	MaxRequests int
	TTL         time.Duration
}

func LoadConfig() (*Config, error) {
	// Try to load .env file
	if err := godotenv.Load(); err != nil {
		// If .env doesn't exist, use environment variables
		fmt.Println("No .env file found, using environment variables")
	}

	config := &Config{
		ServerPort:              getEnv("SERVER_PORT", "8080"),
		RedisHost:               getEnv("REDIS_HOST", "localhost"),
		RedisPort:               getEnv("REDIS_PORT", "6379"),
		MaxRequestsPerSecond:    getEnvAsInt("MAX_REQUESTS_PER_SECOND", 10),
		BlockingTime:            getEnvAsDuration("BLOCKING_TIME_SECONDS", "300"), // 5 minutes default
		EnableIPRateLimiter:     getEnvAsBool("ENABLE_IP_RATE_LIMITER", true),
		EnableTokenRateLimiter:  getEnvAsBool("ENABLE_TOKEN_RATE_LIMITER", true),
		TokenLimits:             make(map[string]TokenLimit),
	}

	// Parse token limits from environment
	// Format: TOKEN_LIMIT_<TOKEN>=MAX_REQUESTS:TTL_SECONDS
	parseTokenLimits(config)

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(name string, defaultValue int) int {
	valueStr := os.Getenv(name)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsBool(name string, defaultValue bool) bool {
	valueStr := os.Getenv(name)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsDuration(name string, defaultValueSeconds string) time.Duration {
	valueStr := os.Getenv(name)
	if valueStr == "" {
		valueStr = defaultValueSeconds
	}
	seconds, err := strconv.Atoi(valueStr)
	if err != nil {
		seconds = 300 // 5 minutes default
	}
	return time.Duration(seconds) * time.Second
}

func parseTokenLimits(config *Config) {
	for _, env := range os.Environ() {
		if len(env) > 12 && env[:12] == "TOKEN_LIMIT_" {
			key := env[:strings.Index(env, "=")]
			value := env[strings.Index(env, "=")+1:]
			
			tokenKey := key[12:] // Remove "TOKEN_LIMIT_" prefix
			
			// Format: MAX_REQUESTS:TTL_SECONDS
			parts := strings.Split(value, ":")
			if len(parts) == 2 {
				maxRequests, err1 := strconv.Atoi(parts[0])
				ttl, err2 := strconv.Atoi(parts[1])
				if err1 == nil && err2 == nil {
					config.TokenLimits[tokenKey] = TokenLimit{
						MaxRequests: maxRequests,
						TTL:         time.Duration(ttl) * time.Second,
					}
				}
			}
		}
	}
}

