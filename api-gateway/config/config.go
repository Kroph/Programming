package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Server struct {
		Port string
	}
	Services struct {
		User struct {
			GrpcURL string
		}
		Inventory struct {
			GrpcURL string
		}
		Order struct {
			GrpcURL string
		}
	}
	Auth struct {
		Secret        string
		ExpiryMinutes int
	}
	Redis struct {
		Addr     string
		Password string
		DB       int
	}
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	config := &Config{}

	config.Server.Port = getEnv("GATEWAY_PORT", "8000")

	config.Services.User.GrpcURL = getEnv("USER_GRPC_URL", "localhost:50053")
	config.Services.Inventory.GrpcURL = getEnv("INVENTORY_GRPC_URL", "localhost:50051")
	config.Services.Order.GrpcURL = getEnv("ORDER_GRPC_URL", "localhost:50052")

	config.Auth.Secret = getEnv("AUTH_SECRET", "your-default-secret-key-change-in-production")
	expiryStr := getEnv("AUTH_EXPIRY_MINUTES", "60")
	expiryMinutes, err := strconv.Atoi(expiryStr)
	if err != nil {
		expiryMinutes = 60
	}
	config.Auth.ExpiryMinutes = expiryMinutes

	// Redis configuration
	config.Redis.Addr = getEnv("REDIS_ADDR", "localhost:6379")
	config.Redis.Password = getEnv("REDIS_PASSWORD", "")

	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "3"))
	if err != nil {
		redisDB = 3
	}
	config.Redis.DB = redisDB

	return config
}

func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}
