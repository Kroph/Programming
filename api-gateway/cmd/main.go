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

	"api-gateway/config"
	"api-gateway/handler"
	"api-gateway/middleware"
	"api-gateway/service"
)

func main() {
	cfg := config.LoadConfig()

	authService := service.NewAuthService(cfg.Auth.Secret, cfg.Auth.ExpiryMinutes)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.RequestID())
	router.Use(middleware.Telemetry())

	protectedRouter := router.Group("/api/v1")
	protectedRouter.Use(middleware.Authentication(authService))

	inventoryService := service.NewProxyService(cfg.Services.Inventory.URL)
	handler.RegisterProxyRoutes(protectedRouter, "/products", inventoryService)
	handler.RegisterProxyRoutes(protectedRouter, "/categories", inventoryService)

	orderService := service.NewProxyService(cfg.Services.Order.URL)
	handler.RegisterProxyRoutes(protectedRouter, "/orders", orderService)

	server := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	go func() {
		log.Printf("API Gateway starting on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

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
