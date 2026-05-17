package observability

import (
	"context"

	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"

type requestIDKey struct{}

func NewRequestID() string {
	return uuid.NewString()
}

func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey{}).(string)
	return requestID
}
