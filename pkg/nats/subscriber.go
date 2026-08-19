package nats

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
)

type MessageHandler func(ctx context.Context, msg jetstream.Msg) error

type Subscriber struct {
	js jetstream.JetStream
}

func NewSubscriber(js jetstream.JetStream) *Subscriber {
	return &Subscriber{js: js}
}

func (s *Subscriber) Subscribe(ctx context.Context, subject string, handler MessageHandler) error {
	cons, err := s.js.CreateOrUpdateConsumer(ctx, "DEVICE_DATA", jetstream.ConsumerConfig{
		FilterSubject: subject,
	})
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}
	iter, err := cons.Fetch(1, jetstream.FetchMaxWait(0))
	if err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}
	for msg := range iter.Messages() {
		if err := handler(ctx, msg); err != nil {
			fmt.Println("[NATS] handler error:", err)
			msg.Nak()
			continue
		}
		msg.Ack()
	}
	fmt.Println("[NATS] subscribed to", subject)
	return nil
}
