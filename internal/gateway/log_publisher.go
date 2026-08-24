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
	fmt.Printf("[LogPub] ⚠️ NATS 不可用，仅日志记录: device=%s domain=%s units=%d type=%d\n",
		env.DeviceID, env.DomainID, len(env.Units), env.Type)
	return nil
}
