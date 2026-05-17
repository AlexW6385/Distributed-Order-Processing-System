package observability

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const requestIDMetadataKey = "x-request-id"

func UnaryClientInterceptor(serviceName string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req any, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		requestID := RequestIDFromContext(ctx)
		if requestID != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, requestIDMetadataKey, requestID)
		}

		startedAt := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		logGRPC(ctx, serviceName, "grpc client request completed", method, startedAt, err)
		return err
	}
}

func UnaryServerInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		requestID := requestIDFromIncomingMetadata(ctx)
		if requestID == "" {
			requestID = NewRequestID()
		}
		ctx = ContextWithRequestID(ctx, requestID)

		startedAt := time.Now()
		resp, err := handler(ctx, req)
		logGRPC(ctx, serviceName, "grpc server request completed", info.FullMethod, startedAt, err)
		return resp, err
	}
}

func requestIDFromIncomingMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(requestIDMetadataKey)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func logGRPC(ctx context.Context, serviceName string, message string, method string, startedAt time.Time, err error) {
	code := codes.OK
	if err != nil {
		code = status.Code(err)
	}

	slog.InfoContext(
		ctx,
		message,
		slog.String("service", serviceName),
		slog.String("request_id", RequestIDFromContext(ctx)),
		slog.String("method", method),
		slog.String("code", code.String()),
		slog.Duration("duration", time.Since(startedAt)),
	)
}
