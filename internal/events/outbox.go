package events

import (
	"context"
	"database/sql"
	"time"
)

const OrderPaidEventType = "order.paid"

type OutboxEvent struct {
	ID            string
	AggregateType string
	AggregateID   string
	EventType     string
	Payload       []byte
	Attempts      int
	CreatedAt     time.Time
}

type OutboxRepository struct {
	db *sql.DB
}

func NewOutboxRepository(db *sql.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) FetchPending(ctx context.Context, limit int) ([]OutboxEvent, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, attempts, created_at
		FROM outbox_events
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]OutboxEvent, 0)
	for rows.Next() {
		var event OutboxEvent
		if err := rows.Scan(
			&event.ID,
			&event.AggregateType,
			&event.AggregateID,
			&event.EventType,
			&event.Payload,
			&event.Attempts,
			&event.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func (r *OutboxRepository) MarkPublished(ctx context.Context, eventID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE outbox_events
		SET status = 'published', published_at = now(), last_error = ''
		WHERE id = $1
	`, eventID)
	return err
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, eventID string, lastError string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE outbox_events
		SET attempts = attempts + 1, last_error = $2
		WHERE id = $1
	`, eventID, lastError)
	return err
}
