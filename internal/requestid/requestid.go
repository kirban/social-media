// Package requestid carries a request-correlation id through a Go context
// on the receiving side of a gRPC call, mirroring what chi's RequestID
// middleware already does for the gateway's own HTTP context.
package requestid

import "context"

type ctxKey struct{}

// WithID returns a context carrying id, retrievable via FromContext.
func WithID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// FromContext returns the request id stored by WithID, or "" if none was set.
func FromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}
