package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/config"
	"github.com/AlexW6385/Distributed-Order-Processing-System/internal/events"
	"github.com/segmentio/kafka-go"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg := config.Load()
	if len(cfg.KafkaBrokers) == 0 {
		log.Fatal("KAFKA_BROKERS is required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := events.EnsureTopic(ctx, cfg.KafkaBrokers, cfg.OrderPaidTopic); err != nil {
		log.Fatalf("ensure kafka topic: %v", err)
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.KafkaBrokers,
		Topic:       cfg.OrderPaidTopic,
		GroupID:     cfg.NotificationGroup,
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: kafka.FirstOffset,
	})
	defer reader.Close()

	slog.Info("notification-service listening", slog.String("topic", cfg.OrderPaidTopic), slog.Any("brokers", cfg.KafkaBrokers))
	for {
		message, err := reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			slog.Error("read kafka message", slog.String("error", err.Error()))
			continue
		}

		var event events.OrderPaidEvent
		if err := json.Unmarshal(message.Value, &event); err != nil {
			slog.Error("decode order paid event", slog.String("error", err.Error()))
			continue
		}

		slog.Info(
			"notification sent",
			slog.String("customer_email", event.CustomerEmail),
			slog.String("order_id", event.OrderID),
			slog.String("payment_id", event.PaymentID),
			slog.Int("amount_cents", event.AmountCents),
		)
	}
}
