package events

import (
	"context"
	"strings"

	"github.com/segmentio/kafka-go"
)

func EnsureTopic(ctx context.Context, brokers []string, topic string) error {
	if len(brokers) == 0 {
		return nil
	}

	conn, err := kafka.DialContext(ctx, "tcp", brokers[0])
	if err != nil {
		return err
	}
	defer conn.Close()

	err = conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "already exists") {
		return err
	}
	return nil
}
