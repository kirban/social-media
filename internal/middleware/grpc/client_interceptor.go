// Package grpcinterceptor propagates the gateway's X-Request-Id across gRPC
// calls to the extracted services, so one client request correlates across
// every container's logs regardless of how many internal hops it takes.
package middleware_grpc

import (
	"context"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// metadataKey is the gRPC metadata key carrying the gateway's request id.
// gRPC lowercases all metadata keys on the wire, so this differs from the
// "X-Request-Id" HTTP header name even though it carries the same value.
const metadataKey = "x-request-id"

// UnaryClientInterceptor reads the request id chimiddleware.RequestID
// already set on ctx (the same one internal/api/errors.go and
// internal/middleware/logging.go read via chimiddleware.GetReqID) and
// forwards it as outgoing gRPC metadata, so UnaryServerInterceptor on the
// receiving service can recover it.
func UnaryClientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	if id := chimiddleware.GetReqID(ctx); id != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, metadataKey, id)
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}
