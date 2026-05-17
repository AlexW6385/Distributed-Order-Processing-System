package events

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"
)

type OutboxStore interface {
	FetchPending(ctx context.Context, limit int) ([]OutboxEvent, error)
	MarkPublished(ctx context.Context, eventID string) error
	MarkFailed(ctx context.Context, eventID string, lastError string) error
}

type OutboxPublisher struct {
	store     OutboxStore
	publisher OrderPaidPublisher
	interval  time.Duration
	batchSize int
}

func NewOutboxPublisher(store OutboxStore, publisher OrderPaidPublisher, interval time.Duration, batchSize int) *OutboxPublisher {
	return &OutboxPublisher{
		store:     store,
		publisher: publisher,
		interval:  interval,
		batchSize: batchSize,
	}
}

func (p *OutboxPublisher) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	p.publishPending(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.publishPending(ctx)
		}
	}
}

func (p *OutboxPublisher) publishPending(ctx context.Context) {
	events, err := p.store.FetchPending(ctx, p.batchSize)
	if err != nil {
		slog.ErrorContext(ctx, "fetch pending outbox events", slog.String("error", err.Error()))
		return
	}

	for _, event := range events {
		if event.EventType != OrderPaidEventType {
			_ = p.store.MarkFailed(ctx, event.ID, "unsupported event type")
			continue
		}

		var orderPaid OrderPaidEvent
		if err := json.Unmarshal(event.Payload, &orderPaid); err != nil {
			_ = p.store.MarkFailed(ctx, event.ID, err.Error())
			continue
		}

		if err := p.publisher.PublishOrderPaid(ctx, orderPaid); err != nil {
			_ = p.store.MarkFailed(ctx, event.ID, err.Error())
			continue
		}

		if err := p.store.MarkPublished(ctx, event.ID); err != nil {
			slog.ErrorContext(ctx, "mark outbox event published", slog.String("event_id", event.ID), slog.String("error", err.Error()))
		}
	}
}
