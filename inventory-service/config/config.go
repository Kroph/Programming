package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DB struct {
		Host     string
		Port     string
		User     string
		Password string
		Name     string
	}
	Server struct {
		Port     string
		GrpcPort string
	}
	NATS struct {
		URL string
	}
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	config := &Config{}

	config.DB.Host = getEnv("DB_HOST", "localhost")
	config.DB.Port = getEnv("DB_PORT", "5432")
	config.DB.User = getEnv("DB_USER", "postgres")
	config.DB.Password = getEnv("DB_PASSWORD", "postgres")
	config.DB.Name = getEnv("DB_NAME", "inventory")

	config.Server.Port = getEnv("INVENTORY_HTTP_PORT", "8080")
	config.Server.GrpcPort = getEnv("INVENTORY_GRPC_PORT", "50051")
	config.NATS.URL = getEnv("NATS_URL", "nats://localhost:4222")

	return config
}

func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}
