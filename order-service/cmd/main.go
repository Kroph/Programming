package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"

	"order-service/config"
	"order-service/internal/handler"
	"order-service/internal/publisher"
	"order-service/internal/repository"
	"order-service/internal/service"

	pb "github.com/Kroph/Programming/proto/order"
	"github.com/Kroph/Programming/shared/pkg/nats"

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

	// Initialize NATS client
	natsClient, err := nats.NewClient(cfg.NATS.URL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsClient.Close()

	// Initialize repositories
	orderRepo := repository.NewPostgresOrderRepository(db)

	// Initialize publisher
	orderPublisher := publisher.NewOrderPublisher(natsClient)

	// Initialize services
	orderService := service.NewOrderService(orderRepo, orderPublisher)

	// Initialize gRPC server
	lis, err := net.Listen("tcp", ":"+cfg.Server.GrpcPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	// Register order service handler
	orderHandler := handler.NewOrderGrpcHandler(orderService)
	pb.RegisterOrderServiceServer(grpcServer, orderHandler)

	// Register payment service handler
	paymentHandler := handler.NewPaymentGrpcHandler()
	pb.RegisterPaymentServiceServer(grpcServer, paymentHandler)

	// Enable reflection for tools like grpcurl
	reflection.Register(grpcServer)

	log.Printf("Order Service gRPC server starting on port %s", cfg.Server.GrpcPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
