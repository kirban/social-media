package broker

import (
	"context"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/test"

	"github.com/kirban/social-media/internal/config"
	"github.com/kirban/social-media/internal/model"
)

// startJetStream boots an in-process NATS server with JetStream enabled and
// returns its client URL. It shuts down when the test ends.
func startJetStream(t *testing.T) string {
	t.Helper()
	opts := natsserver.DefaultTestOptions
	opts.Port = -1 // random free port
	opts.JetStream = true
	opts.StoreDir = t.TempDir()

	srv := natsserver.RunServer(&opts)
	t.Cleanup(srv.Shutdown)
	return srv.ClientURL()
}

func testConfig(url string) config.NATSConfig {
	return config.NATSConfig{
		URL:            url,
		Stream:         "POSTS",
		SubjectPrefix:  "posts",
		MaxAge:         time.Minute,
		PublishRetries: 3,
	}
}

// TestPublishConsumeFanout exercises the whole pipeline end to end against a
// real (embedded) JetStream: publish a post.created event, and assert the
// consumer's fan-out invalidates the follower's cache and notifies them.
func TestPublishConsumeFanout(t *testing.T) {
	cfg := testConfig(startJetStream(t))

	nc, js, err := Connect(cfg)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer nc.Close()

	stream, err := EnsureStream(context.Background(), js, cfg)
	if err != nil {
		t.Fatalf("ensure stream: %v", err)
	}

	cache := &fakeCache{}
	hub := newFakeHub()
	fanout := NewFeedFanout(fakeFollowers{list: []string{"follower1"}}, cache, hub, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumer := NewConsumer(stream, cfg, fanout, testLogger())
	consumerReady := make(chan struct{})
	go func() {
		close(consumerReady)
		_ = consumer.Run(ctx)
	}()
	<-consumerReady
	// The ordered consumer delivers from DeliverNew, so give it a moment to
	// establish its subscription before we publish.
	time.Sleep(200 * time.Millisecond)

	pub := NewPublisher(js, cfg, testLogger())
	if err := pub.PublishPostCreated(context.Background(), model.PostCreatedEvent{
		PostID:   "p1",
		AuthorID: "author",
		Text:     "hi",
	}); err != nil {
		t.Fatalf("publish: %v", err)
	}

	waitFor(t, 3*time.Second, func() bool {
		hub.mu.Lock()
		defer hub.mu.Unlock()
		return hub.sent["follower1"] == 1
	}, "follower to receive WS notification")

	cache.mu.Lock()
	defer cache.mu.Unlock()
	if !contains(cache.deleted, model.FeedCacheKey("follower1")) {
		t.Errorf("expected follower1's feed cache to be invalidated, deleted=%v", cache.deleted)
	}
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool, what string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}
