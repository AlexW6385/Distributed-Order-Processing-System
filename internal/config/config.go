package config

import "os"

type Config struct {
	DatabaseURL string
	Port        string
}

func Load() Config {
	return Config{
		DatabaseURL: getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/distributed_order_processing_system?sslmode=disable"),
		Port:        getenv("PORT", "8080"),
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
