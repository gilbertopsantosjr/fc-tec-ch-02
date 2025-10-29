package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"fc-tec-ch-02/internal/limiter"
)

// RateLimitMiddleware creates a middleware that enforces rate limiting
func RateLimitMiddleware(rateLimiterService *limiter.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			
			// Extract IP address
			ip := getClientIP(r)
			
			// Extract token from header (check X-API-Token or Authorization header)
			token := getTokenFromRequest(r)
			
			// Check rate limit and increment
			allowed, resetTime, err := rateLimiterService.CheckAndIncrement(ctx, ip, token)
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			
			if !allowed {
				// Rate limit exceeded
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-RateLimit-Reset", resetTime.Format(time.RFC3339))
				w.WriteHeader(http.StatusTooManyRequests)
				
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":      "Rate limit exceeded",
					"reset_time": resetTime.Format(time.RFC3339),
				})
				return
			}
			
			// Set rate limit headers
			w.Header().Set("X-RateLimit-Reset", resetTime.Format(time.RFC3339))
			
			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies/load balancers)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}
	
	// Check X-Real-IP header (another common proxy header)
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	
	// Fallback to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	
	return ip
}

// getTokenFromRequest extracts the API token from the request headers
func getTokenFromRequest(r *http.Request) string {
	// Check X-API-Token header first
	token := r.Header.Get("X-API-Token")
	if token != "" {
		return token
	}
	
	// Check Authorization header (Bearer token)
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	
	// No token found
	return ""
}

