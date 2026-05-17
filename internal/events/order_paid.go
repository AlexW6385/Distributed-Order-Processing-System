package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
)

type OrderPaidEvent struct {
	EventID       string    `json:"event_id"`
	OrderID       string    `json:"order_id"`
	PaymentID     string    `json:"payment_id"`
	CustomerEmail string    `json:"customer_email"`
	AmountCents   int       `json:"amount_cents"`
	PaidAt        time.Time `json:"paid_at"`
}

type OrderPaidPublisher interface {
	PublishOrderPaid(ctx context.Context, event OrderPaidEvent) error
}

type KafkaOrderPaidPublisher struct {
	writer *kafka.Writer
}

func NewKafkaOrderPaidPublisher(brokers []string, topic string) *KafkaOrderPaidPublisher {
	return &KafkaOrderPaidPublisher{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  topic,
			AllowAutoTopicCreation: true,
		},
	}
}

func (p *KafkaOrderPaidPublisher) PublishOrderPaid(ctx context.Context, event OrderPaidEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.OrderID),
		Value: payload,
	})
}

func (p *KafkaOrderPaidPublisher) Close() error {
	return p.writer.Close()
}
