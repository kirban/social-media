package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/kirban/social-media/internal/config"
	"github.com/kirban/social-media/internal/logger"
	"github.com/kirban/social-media/internal/model"
)

// Publisher publishes post events to JetStream. It is the concrete
// implementation of the service-side event-publisher interface; the service
// depends on that interface, not on this type.
type Publisher struct {
	js  jetstream.JetStream
	cfg config.NATSConfig
	log *logger.AppLogger
}

func NewPublisher(js jetstream.JetStream, cfg config.NATSConfig, log *logger.AppLogger) *Publisher {
	return &Publisher{js: js, cfg: cfg, log: log}
}

func (p *Publisher) PublishPostCreated(ctx context.Context, e model.PostCreatedEvent) error {
	return p.publish(ctx, subjectCreated(p.cfg), e)
}

func (p *Publisher) PublishPostDeleted(ctx context.Context, e model.PostDeletedEvent) error {
	return p.publish(ctx, subjectDeleted(p.cfg), e)
}

// publish marshals payload and publishes it, retrying on transient failures up
// to cfg.PublishRetries with a short linear backoff. Called synchronously after
// the DB commit; a returned error means the event was not durably accepted.
func (p *Publisher) publish(ctx context.Context, subject string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s event: %w", subject, err)
	}

	var lastErr error
	for attempt := 0; attempt <= p.cfg.PublishRetries; attempt++ {
		if _, err := p.js.Publish(ctx, subject, data); err == nil {
			return nil
		} else {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 50 * time.Millisecond):
		}
	}

	return fmt.Errorf("publish %s after %d retries: %w", subject, p.cfg.PublishRetries, lastErr)
}
