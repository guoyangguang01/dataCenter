package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
)

type Publisher struct {
	js jetstream.JetStream
}

func NewPublisher(js jetstream.JetStream) *Publisher {
	return &Publisher{js: js}
}

func (p *Publisher) PublishJSON(ctx context.Context, subject string, data interface{}) error {
	msg, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	ack, err := p.js.Publish(ctx, subject, msg)
	if err != nil {
		return fmt.Errorf("failed to publish to %s: %w", subject, err)
	}
	fmt.Println("[NATS] published to", subject, "seq=", ack.Sequence)
	return nil
}

func (p *Publisher) PublishBytes(ctx context.Context, subject string, data []byte) error {
	ack, err := p.js.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish to %s: %w", subject, err)
	}
	fmt.Println("[NATS] published to", subject, "seq=", ack.Sequence)
	return nil
}
