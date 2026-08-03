package broker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/kirban/social-media/internal/config"
	"github.com/kirban/social-media/internal/logger"
	"github.com/kirban/social-media/internal/model"
)

// Consumer subscribes this instance to the POSTS stream and dispatches each
// event to the fan-out handler. It uses an ephemeral ordered consumer starting
// from DeliverNew: every instance gets its own consumer (broadcast), and a
// restarting instance — which holds no cache or connections yet — simply
// resumes from new events rather than replaying the buffer.
type Consumer struct {
	stream jetstream.Stream
	cfg    config.NATSConfig
	fanout *FeedFanout
	log    *logger.AppLogger
}

func NewConsumer(stream jetstream.Stream, cfg config.NATSConfig, fanout *FeedFanout, log *logger.AppLogger) *Consumer {
	return &Consumer{stream: stream, cfg: cfg, fanout: fanout, log: log}
}

// Run starts consuming and blocks until ctx is cancelled, then stops cleanly.
func (c *Consumer) Run(ctx context.Context) error {
	cons, err := c.stream.OrderedConsumer(ctx, jetstream.OrderedConsumerConfig{
		FilterSubjects: []string{subjectAll(c.cfg)},
		DeliverPolicy:  jetstream.DeliverNewPolicy,
	})
	if err != nil {
		return fmt.Errorf("create ordered consumer: %w", err)
	}

	cc, err := cons.Consume(func(msg jetstream.Msg) {
		c.dispatch(ctx, msg)
	})
	if err != nil {
		return fmt.Errorf("start consume: %w", err)
	}
	defer cc.Stop()

	c.log.Info().Str("stream", c.cfg.Stream).Msg("feed fan-out consumer started")
	<-ctx.Done()
	return nil
}

// dispatch routes a message by subject to the matching handler. Ordered
// consumers auto-advance, so on a handler error we log and move on; the feed
// cache TTL is the safety net that heals any missed invalidation.
func (c *Consumer) dispatch(ctx context.Context, msg jetstream.Msg) {
	switch msg.Subject() {
	case subjectCreated(c.cfg):
		var e model.PostCreatedEvent
		if err := json.Unmarshal(msg.Data(), &e); err != nil {
			c.log.Error().Err(err).Msg("fanout: bad post.created payload")
			return
		}
		if err := c.fanout.HandlePostCreated(ctx, e); err != nil {
			c.log.Error().Err(err).Str("postID", e.PostID).Msg("fanout: handle post.created failed")
		}

	case subjectDeleted(c.cfg):
		var e model.PostDeletedEvent
		if err := json.Unmarshal(msg.Data(), &e); err != nil {
			c.log.Error().Err(err).Msg("fanout: bad post.deleted payload")
			return
		}
		if err := c.fanout.HandlePostDeleted(ctx, e); err != nil {
			c.log.Error().Err(err).Str("postID", e.PostID).Msg("fanout: handle post.deleted failed")
		}

	default:
		c.log.Warn().Str("subject", msg.Subject()).Msg("fanout: unknown subject")
	}
}
