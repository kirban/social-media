package requestid

import (
	"context"
	"testing"
)

func TestWithID_FromContext_RoundTrip(t *testing.T) {
	ctx := WithID(context.Background(), "abc-123")

	if got := FromContext(ctx); got != "abc-123" {
		t.Fatalf("FromContext() = %q, want %q", got, "abc-123")
	}
}

func TestFromContext_Empty(t *testing.T) {
	if got := FromContext(context.Background()); got != "" {
		t.Fatalf("FromContext() = %q, want empty string", got)
	}
}
