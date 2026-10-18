package middleware_grpc

import (
	"context"
	"testing"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func captureInvoker(capturedCtx *context.Context) grpc.UnaryInvoker {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		*capturedCtx = ctx
		return nil
	}
}

func TestUnaryClientInterceptor_SetsOutgoingMetadataFromChiRequestID(t *testing.T) {
	ctx := context.WithValue(context.Background(), chimiddleware.RequestIDKey, "req-42")

	var captured context.Context
	err := UnaryClientInterceptor(ctx, "/svc/Method", nil, nil, nil, captureInvoker(&captured))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	md, ok := metadata.FromOutgoingContext(captured)
	if !ok {
		t.Fatal("expected outgoing metadata to be set")
	}

	values := md.Get(metadataKey)
	if len(values) != 1 || values[0] != "req-42" {
		t.Fatalf("metadata[%q] = %v, want [%q]", metadataKey, values, "req-42")
	}
}

func TestUnaryClientInterceptor_NoRequestID_NoMetadata(t *testing.T) {
	ctx := context.Background()

	var captured context.Context
	err := UnaryClientInterceptor(ctx, "/svc/Method", nil, nil, nil, captureInvoker(&captured))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := metadata.FromOutgoingContext(captured); ok {
		t.Fatal("expected no outgoing metadata to be set when chi request id is empty")
	}
}
