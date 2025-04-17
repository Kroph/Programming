package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"

	"github.com/Kroph/Programming/inventory-service/config"
	"github.com/Kroph/Programming/inventory-service/internal/handler"
	"github.com/Kroph/Programming/inventory-service/internal/repository"
	"github.com/Kroph/Programming/inventory-service/internal/service"
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

	log.Printf("Inventory Service gRPC server starting on port %s", cfg.Server.GrpcPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
