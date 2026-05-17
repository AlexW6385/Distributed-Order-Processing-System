package config

import (
	"os"
	"strings"
)

type Config struct {
	DatabaseURL        string
	Port               string
	RedisAddr          string
	GRPCPort           string
	ProductServiceAddr string
	PaymentServiceAddr string
	KafkaBrokers       []string
	OrderPaidTopic     string
	NotificationGroup  string
}

func Load() Config {
	return Config{
		DatabaseURL:        getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/distributed_order_processing_system?sslmode=disable"),
		Port:               getenv("PORT", "8080"),
		RedisAddr:          getenv("REDIS_ADDR", "localhost:6379"),
		GRPCPort:           getenv("GRPC_PORT", "50051"),
		ProductServiceAddr: getenv("PRODUCT_SERVICE_ADDR", "localhost:50051"),
		PaymentServiceAddr: getenv("PAYMENT_SERVICE_ADDR", "localhost:50052"),
		KafkaBrokers:       splitCSV(getenv("KAFKA_BROKERS", "")),
		OrderPaidTopic:     getenv("ORDER_PAID_TOPIC", "order.paid"),
		NotificationGroup:  getenv("NOTIFICATION_GROUP", "notification-service"),
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}
