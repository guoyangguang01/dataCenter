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

// Subscribe 订阅 DEVICE_DATA 流中的指定主题（保持向后兼容）
func (s *Subscriber) Subscribe(ctx context.Context, subject string, handler MessageHandler) error {
	return s.SubscribeToStream(ctx, "DEVICE_DATA", subject, handler)
}

// SubscribeToStream 订阅指定流中的主题，持续消费消息直到 ctx 取消
func (s *Subscriber) SubscribeToStream(ctx context.Context, stream, subject string, handler MessageHandler) error {
	cons, err := s.js.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
		FilterSubject: subject,
	})
	if err != nil {
		return fmt.Errorf("failed to create consumer on stream %s: %w", stream, err)
	}

	// 使用 Messages() 进行持续消费（而非 Fetch 的单次拉取）
	iter, err := cons.Messages()
	if err != nil {
		return fmt.Errorf("failed to get message iterator: %w", err)
	}

	go func() {
		for {
			msg, err := iter.Next()
			if err != nil {
				// ctx 取消时 iter.Next() 会返回错误
				return
			}
			if err := handler(ctx, msg); err != nil {
				fmt.Printf("[NATS] handler error on %s: %v\n", subject, err)
				msg.Nak()
				continue
			}
			msg.Ack()
		}
	}()

	fmt.Printf("[NATS] subscribed to %s on stream %s\n", subject, stream)
	return nil
}

// Stop 停止所有订阅（通过取消传入的 ctx 实现）
func (s *Subscriber) Stop() {
	fmt.Println("[NATS] subscriber stopping")
}
