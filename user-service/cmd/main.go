package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"

	pb "github.com/Kroph/Programming/proto/user"
	"github.com/Kroph/Programming/user-service/config"
	"github.com/Kroph/Programming/user-service/internal/handler"
	"github.com/Kroph/Programming/user-service/internal/repository"
	"github.com/Kroph/Programming/user-service/internal/service"

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
	userRepo := repository.NewPostgresUserRepository(db)

	// Initialize services
	userService := service.NewUserService(userRepo, cfg.Auth.Secret, cfg.Auth.ExpiryMinutes)

	// Initialize gRPC server
	lis, err := net.Listen("tcp", ":"+cfg.Server.GrpcPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	// Register user service handler
	userHandler := handler.NewUserGrpcHandler(userService)
	pb.RegisterUserServiceServer(grpcServer, userHandler)

	// Enable reflection for tools like grpcurl
	reflection.Register(grpcServer)

	log.Printf("User Service gRPC server starting on port %s", cfg.Server.GrpcPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
