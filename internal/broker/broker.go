// Package broker wires post events onto a NATS JetStream stream and fans them
// out to feed consumers. The stream (POSTS) is a short-lived buffer for
// post.created / post.deleted events, not a system of record: every API
// instance runs its own ephemeral consumer, so each keeps its in-process feed
// cache and WebSocket connections up to date independently.
package broker

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/kirban/social-media/internal/config"
)

// Subjects derived from the configured prefix (default "posts").
func subjectCreated(cfg config.NATSConfig) string { return cfg.SubjectPrefix + ".created" }
func subjectDeleted(cfg config.NATSConfig) string { return cfg.SubjectPrefix + ".deleted" }
func subjectAll(cfg config.NATSConfig) string     { return cfg.SubjectPrefix + ".*" }

// Connect dials NATS and returns a JetStream context. The caller owns the
// connection and must Close it on shutdown.
func Connect(cfg config.NATSConfig) (*nats.Conn, jetstream.JetStream, error) {
	nc, err := nats.Connect(cfg.URL,
		nats.Name("social-media"),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("connect nats %q: %w", cfg.URL, err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("jetstream context: %w", err)
	}

	return nc, js, nil
}

// EnsureStream idempotently creates or updates the POSTS stream that buffers
// post events. Retention is limits-based with a bounded MaxAge — messages are
// dropped once consumed by every live instance or once they age out.
func EnsureStream(ctx context.Context, js jetstream.JetStream, cfg config.NATSConfig) (jetstream.Stream, error) {
	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      cfg.Stream,
		Subjects:  []string{subjectAll(cfg)},
		Retention: jetstream.LimitsPolicy,
		Storage:   jetstream.FileStorage,
		MaxAge:    cfg.MaxAge,
	})
	if err != nil {
		return nil, fmt.Errorf("ensure stream %q: %w", cfg.Stream, err)
	}
	return stream, nil
}
