package middleware_grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	applogger "github.com/kirban/social-media/internal/logger"
	"github.com/kirban/social-media/internal/requestid"
)

// UnaryServerInterceptor reads the x-request-id metadata UnaryClientInterceptor
// sets and stores it in context via internal/requestid, so handlers and
// LoggingUnaryServerInterceptor on this service can recover the gateway's
// request id.
func UnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(metadataKey); len(values) > 0 {
			ctx = requestid.WithID(ctx, values[0])
		}
	}
	return handler(ctx, req)
}

// LoggingUnaryServerInterceptor logs one line per RPC under the same
// request_id field internal/middleware.Logging already uses for the
// gateway's HTTP requests, so gateway and service logs for one call
// correlate under `docker compose logs` or any log aggregation.
//
// It must run after UnaryServerInterceptor in the chain so requestid.FromContext
// sees the id that interceptor already extracted.
func LoggingUnaryServerInterceptor(logger *applogger.AppLogger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)

		logger.Info().
			Str("request_id", requestid.FromContext(ctx)).
			Str("rpc", info.FullMethod).
			Str("code", status.Code(err).String()).
			Dur("latency", time.Since(start)).
			Msg("rpc")

		return resp, err
	}
}
