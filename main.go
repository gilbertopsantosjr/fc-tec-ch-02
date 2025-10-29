package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fc-tec-ch-02/internal/config"
	"fc-tec-ch-02/internal/handlers"
	"fc-tec-ch-02/internal/limiter"
	"fc-tec-ch-02/internal/middleware"
	"fc-tec-ch-02/internal/storage"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize storage (Redis)
	storageInstance, err := storage.NewRedisStorage(cfg.RedisHost, cfg.RedisPort)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer storageInstance.Close()

	// Test storage connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := storageInstance.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping Redis: %v", err)
	}
	log.Println("Successfully connected to Redis")

	// Initialize rate limiter service
	rateLimiterService := limiter.NewService(storageInstance, cfg)

	// Setup routes
	mux := http.NewServeMux()

	// Test endpoint
	mux.HandleFunc("/health", handlers.HealthHandler)
	mux.HandleFunc("/test", handlers.TestHandler)

	// Create server with middleware
	handler := middleware.RateLimitMiddleware(rateLimiterService)(mux)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.ServerPort),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", cfg.ServerPort)
		log.Printf("Rate limiter configured: IP=%v, Token=%v", cfg.EnableIPRateLimiter, cfg.EnableTokenRateLimiter)
		log.Printf("Max requests per second: %d", cfg.MaxRequestsPerSecond)
		log.Printf("Blocking time: %v", cfg.BlockingTime)
		
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited successfully")
}


