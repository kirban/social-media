package middleware_grpc

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	applogger "github.com/kirban/social-media/internal/logger"
	"github.com/kirban/social-media/internal/requestid"
)

func TestUnaryServerInterceptor_ExtractsRequestIDFromIncomingMetadata(t *testing.T) {
	md := metadata.New(map[string]string{metadataKey: "req-42"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	var gotID string
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		gotID = requestid.FromContext(ctx)
		return "resp", nil
	}

	resp, err := UnaryServerInterceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "resp" {
		t.Fatalf("resp = %v, want %q", resp, "resp")
	}
	if gotID != "req-42" {
		t.Fatalf("requestid.FromContext() = %q, want %q", gotID, "req-42")
	}
}

func TestUnaryServerInterceptor_NoMetadata_EmptyID(t *testing.T) {
	var gotID string
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		gotID = requestid.FromContext(ctx)
		return nil, nil
	}

	_, err := UnaryServerInterceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotID != "" {
		t.Fatalf("requestid.FromContext() = %q, want empty", gotID)
	}
}

func TestLoggingUnaryServerInterceptor_PassesThroughResponseAndError(t *testing.T) {
	logger := &applogger.AppLogger{Logger: zerolog.New(io.Discard)}
	wantErr := errors.New("boom")

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, wantErr
	}

	resp, err := LoggingUnaryServerInterceptor(logger)(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, handler)
	if resp != nil {
		t.Fatalf("resp = %v, want nil", resp)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
}
