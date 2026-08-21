package gateway

import (
	"fmt"

	"github.com/datacenter/internal/message"
)

// LogPublisher 简单的日志 Publisher，用于测试（不依赖 NATS）
type LogPublisher struct{}

func NewLogPublisher() *LogPublisher {
	return &LogPublisher{}
}

func (p *LogPublisher) PublishEnvelope(env *message.DeviceEnvelope) error {
	fmt.Printf("[LogPublisher] device=%s domain=%s topic=%s type=%d\n", env.DeviceID, env.DomainID, env.Units[0].Topic, env.Type)
	return nil
}
