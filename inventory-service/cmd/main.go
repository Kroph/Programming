// inventory-service/cmd/main.go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"inventory-service/config"
	"inventory-service/internal/consumer"
	"inventory-service/internal/handler"
	"inventory-service/internal/repository"
	"inventory-service/internal/service"
	"shared/pkg/nats"

	pb "github.com/Kroph/Programming/proto/inventory"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.LoadConfig()

	// Connect to database
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DB.Host,
		cfg.DB.Port,
		cfg.DB.User,
		cfg.DB.Password,
		cfg.DB.Name,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Check DB connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Initialize repositories
	productRepo := repository.NewPostgresProductRepository(db)
	categoryRepo := repository.NewPostgresCategoryRepository(db)

	// Initialize services
	productService := service.NewProductService(productRepo)
	categoryService := service.NewCategoryService(categoryRepo)

	// Initialize NATS client
	natsClient, err := nats.NewClient(cfg.NATS.URL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsClient.Close()

	// Initialize consumer
	orderConsumer := consumer.NewOrderConsumer(natsClient, productService)

	// Start consumer in a goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := orderConsumer.Start(ctx); err != nil {
			log.Printf("Consumer error: %v", err)
		}
	}()

	// Initialize gRPC server
	lis, err := net.Listen("tcp", ":"+cfg.Server.GrpcPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	// Register product service handler
	productHandler := handler.NewProductGrpcHandler(productService)
	pb.RegisterProductServiceServer(grpcServer, productHandler)

	// Register category service handler
	categoryHandler := handler.NewCategoryGrpcHandler(categoryService)
	pb.RegisterCategoryServiceServer(grpcServer, categoryHandler)

	// Enable reflection for tools like grpcurl
	reflection.Register(grpcServer)

	// Start gRPC server in a goroutine
	go func() {
		log.Printf("Inventory Service gRPC server starting on port %s", cfg.Server.GrpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	cancel()
}
