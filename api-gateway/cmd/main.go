package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Kroph/Programming/api-gateway/config"
	"github.com/Kroph/Programming/api-gateway/handler"
	"github.com/Kroph/Programming/api-gateway/middleware"
	"github.com/Kroph/Programming/api-gateway/service"
)

func main() {
	cfg := config.LoadConfig()

	// Initialize gRPC clients
	grpcClients, err := service.NewGrpcClients(
		cfg.Services.User.GrpcURL,
		cfg.Services.Inventory.GrpcURL,
		cfg.Services.Order.GrpcURL,
	)
	if err != nil {
		log.Fatalf("Failed to initialize gRPC clients: %v", err)
	}

	// Initialize auth service
	authService := service.NewAuthService(cfg.Auth.Secret, cfg.Auth.ExpiryMinutes)

	// Initialize handler
	h := handler.NewHandler(grpcClients, authService)

	// Set up router with middleware
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.RequestID())
	router.Use(middleware.Telemetry())

	// Register routes
	handler.RegisterRoutes(router, h)

	// Initialize HTTP server
	server := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("API Gateway starting on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
