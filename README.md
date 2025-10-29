# Rate Limiter - Go Challenge

A production-ready rate limiter implementation in Go that limits requests per second based on IP addresses or access tokens. The rate limiter uses Redis for distributed state management and follows the strategy pattern for easy extensibility.

## Features

- ✅ **IP-based Rate Limiting**: Limit requests per IP address
- ✅ **Token-based Rate Limiting**: Limit requests per API key/token
- ✅ **Token Overrides IP**: When a token is provided, it overrides IP limits
- ✅ **Configurable Limits**: Set different limits per token
- ✅ **Blocking Period**: Configurable blocking time when limit is exceeded
- ✅ **Strategy Pattern**: Easy to switch from Redis to other storage backends
- ✅ **HTTP 429 Response**: Proper response when rate limit is exceeded
- ✅ **Redis Integration**: Uses Redis for distributed rate limiting state
- ✅ **Docker Support**: Ready-to-run Docker Compose setup
- ✅ **Comprehensive Tests**: Unit tests included

## Architecture

```
┌─────────────┐
│   HTTP      │
│  Request    │
└──────┬──────┘
       │
       v
┌─────────────────┐
│  Rate Limit     │
│  Middleware     │
└────────┬────────┘
         │
         v
┌─────────────────┐
│ Rate Limiter    │
│    Service      │
└────────┬────────┘
         │
         v
┌─────────────────┐      ┌──────────────┐
│  Storage        │─────>│  Redis       │
│  (Interface)    │      │  (Adapter)   │
└─────────────────┘      └──────────────┘
```

### Components

1. **Middleware** (`internal/middleware/ratelimit.go`): HTTP middleware that intercepts requests
2. **Service** (`internal/limiter/service.go`): Business logic for rate limiting
3. **Limiter** (`internal/limiter/limiter.go`): Core rate limiting logic
4. **Storage Interface** (`internal/storage/storage.go`): Abstraction for storage
5. **Redis Adapter** (`internal/storage/redis_storage.go`): Redis implementation

## Configuration

Configuration is done via environment variables or a `.env` file:

| Variable                    | Default     | Description                                                |
| --------------------------- | ----------- | ---------------------------------------------------------- |
| `SERVER_PORT`               | `8080`      | Server port                                                |
| `REDIS_HOST`                | `localhost` | Redis host                                                 |
| `REDIS_PORT`                | `6379`      | Redis port                                                 |
| `MAX_REQUESTS_PER_SECOND`   | `10`        | Max requests per IP                                        |
| `BLOCKING_TIME_SECONDS`     | `300`       | Blocking time in seconds (5 minutes)                       |
| `ENABLE_IP_RATE_LIMITER`    | `true`      | Enable IP-based rate limiting                              |
| `ENABLE_TOKEN_RATE_LIMITER` | `true`      | Enable token-based rate limiting                           |
| `TOKEN_LIMIT_<TOKEN>`       | -           | Token-specific limits (format: `MAX_REQUESTS:TTL_SECONDS`) |

### Example Configuration

```env
SERVER_PORT=8080
REDIS_HOST=redis
REDIS_PORT=6379
MAX_REQUESTS_PER_SECOND=5
BLOCKING_TIME_SECONDS=300
ENABLE_IP_RATE_LIMITER=true
ENABLE_TOKEN_RATE_LIMITER=true

# Token limits
TOKEN_LIMIT_my-secret-token=10:60
TOKEN_LIMIT_premium-token=100:300
```

## Usage

### Quick Start with Docker Compose

```bash
# Start all services (app + Redis)
docker-compose up -d

# View logs
docker-compose logs -f app

# Stop services
docker-compose down
```

### Development

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Run application (requires Redis running)
go run main.go

# Build
go build -o bin/main .
```

### Using Make

```bash
# Build
make build

# Run
make run

# Run tests
make test

# Docker commands
make docker-up
make docker-down
make docker-logs

# Test rate limiter
make test-request
make test-token
```

## API Usage

### Endpoints

- `GET /health` - Health check endpoint
- `GET /test` - Test endpoint protected by rate limiter

### Making Requests

#### Without Token (IP-based limiting)

```bash
# Request will be rate-limited by IP
curl http://localhost:8080/test
```

#### With Token (token limits override IP)

```bash
# Use API_KEY header with token
curl -H "API_KEY: my-secret-token" http://localhost:8080/test

# Or with explicit format
curl -H "API_KEY: my-secret-token" http://localhost:8080/test
```

### Rate Limit Headers

The rate limiter returns the following headers:

- `X-RateLimit-Limit`: The rate limit that was applied
- `X-RateLimit-Remaining`: Remaining requests (when approaching limit)
- `X-RateLimit-Reset`: Time when the limit resets
- `Retry-After`: Seconds until retry is allowed

### Error Response

When rate limit is exceeded:

**HTTP Status**: `429 Too Many Requests`

```json
{
  "message": "you have reached the maximum number of requests or actions allowed within a certain time frame"
}
```

## Examples

### Example 1: IP Rate Limiting (5 req/s, blocked for 5 minutes)

Configure:

```env
MAX_REQUESTS_PER_SECOND=5
BLOCKING_TIME_SECONDS=300
```

Test:

```bash
# First 5 requests succeed
curl http://localhost:8080/test
curl http://localhost:8080/test
curl http://localhost:8080/test
curl http://localhost:8080/test
curl http://localhost:8080/test

# 6th request blocked (HTTP 429)
curl http://localhost:8080/test
```

### Example 2: Token Rate Limiting

Configure:

```env
TOKEN_LIMIT_my-token=10:60
```

Test:

```bash
# 10 requests with token succeed
for i in {1..10}; do
  curl -H "API_KEY: my-token" http://localhost:8080/test
done

# 11th request blocked
curl -H "API_KEY: my-token" http://localhost:8080/test
```

### Example 3: Token Overrides IP Limit

Configure:

```env
MAX_REQUESTS_PER_SECOND=5  # Low IP limit
TOKEN_LIMIT_premium=100:300 # High token limit
```

Test:

```bash
# Without token: limited to 5 req/s
curl http://localhost:8080/test  # Blocked after 5

# With token: limited to 100 req/s
curl -H "API_KEY: premium" http://localhost:8080/test  # Allowed up to 100
```

## Testing

Run all tests:

```bash
go test ./...
```

Run with verbose output:

```bash
go test -v ./...
```

Test specific package:

```bash
go test ./internal/limiter/...
```

## Project Structure

```
fc-tec-ch-02/
├── internal/
│   ├── config/          # Configuration management
│   ├── handlers/        # HTTP handlers
│   ├── limiter/         # Rate limiting logic
│   ├── middleware/      # HTTP middleware
│   └── storage/         # Storage interface & implementations
├── main.go              # Application entry point
├── Dockerfile           # Docker build instructions
├── docker-compose.yml   # Docker Compose setup
├── Makefile            # Build automation
└── README.md           # This file
```

## How It Works

1. **Request Arrives**: HTTP request hits the middleware
2. **Extract Identifier**: Middleware extracts IP address and/or token
3. **Check Limit**: Service checks if request is allowed
4. **Consult Storage**: Current count and reset time retrieved from Redis
5. **Decision**:
   - If allowed: request proceeds and counter incremented
   - If blocked: return HTTP 429 with reset time
6. **Update Storage**: Increment counter with TTL in Redis

### Token Priority

When both IP and token are present:

1. Token limits take priority over IP limits
2. Token counter is incremented
3. IP counter is not incremented when token is present

## Strategy Pattern

The rate limiter uses the strategy pattern for storage abstraction:

```go
type Storage interface {
    Increment(ctx context.Context, key string, ttl time.Duration) (int, time.Time, error)
    Get(ctx context.Context, key string) (*RateLimitInfo, error)
    // ... more methods
}
```

Current implementation: `RedisStorage`

To add a new storage backend (e.g., Memcached, In-memory):

1. Implement the `Storage` interface
2. Update factory in `main.go`
3. No other code changes needed

## Performance Considerations

- **Redis**: Fast, distributed, persistent
- **TTL**: Automatic expiration of rate limit counters
- **Atomic Operations**: Redis INCR provides atomic increments
- **Minimal Overhead**: Middleware adds minimal latency
- **Concurrent Safe**: Redis handles concurrent requests

## License

This project is part of a FullCycle challenge.

## Author

Developed as part of FullCycle Go challenge.

