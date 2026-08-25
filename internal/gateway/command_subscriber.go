package gateway

import (
	"context"
	"fmt"
	"strings"

	"github.com/datacenter/pkg/nats"
	"github.com/nats-io/nats.go/jetstream"
)

// CommandSubscriber 订阅 NATS DEVICE_COMMAND 流，将下行命令路由到对应网关
type CommandSubscriber struct {
	subscriber *nats.Subscriber
	publisher  CommandPublisher
}

// NewCommandSubscriber 创建命令订阅者
func NewCommandSubscriber(subscriber *nats.Subscriber, publisher CommandPublisher) *CommandSubscriber {
	return &CommandSubscriber{
		subscriber: subscriber,
		publisher:  publisher,
	}
}

// Start 开始订阅 DEVICE_COMMAND 流
// 主题格式: domains.{domainID}.devices.{region}.{deviceType}.{deviceID}.down
func (cs *CommandSubscriber) Start(ctx context.Context) error {
	return cs.subscriber.SubscribeToStream(ctx, "DEVICE_COMMAND", "domains.*.devices.*.*.*.down", cs.handleCommand)
}

// handleCommand 处理从 NATS 收到的下行命令
func (cs *CommandSubscriber) handleCommand(ctx context.Context, msg jetstream.Msg) error {
	subject := msg.Subject()
	data := msg.Data()

	// 从主题解析 deviceID
	// 格式: domains.{domainID}.devices.{region}.{deviceType}.{deviceID}.down
	deviceID, err := parseDeviceIDFromSubject(subject)
	if err != nil {
		fmt.Printf("[CmdSub] ❌ 无法解析主题: %s error=%v\n", subject, err)
		return err // 返回错误会导致 Nak，消息会被重试
	}

	fmt.Printf("[CmdSub] 📨 收到命令: device=%s subject=%s payload=%d bytes\n", deviceID, subject, len(data))

	// 路由到设备所在的网关
	if err := cs.publisher.SendToDevice(deviceID, data); err != nil {
		fmt.Printf("[CmdSub] ❌ 命令下发失败: device=%s error=%v\n", deviceID, err)
		return err
	}

	fmt.Printf("[CmdSub] ✅ 命令已下发: device=%s\n", deviceID)
	return nil
}

// parseDeviceIDFromSubject 从 NATS 主题中解析 deviceID
// 主题格式: domains.{domainID}.devices.{region}.{deviceType}.{deviceID}.down
func parseDeviceIDFromSubject(subject string) (string, error) {
	parts := strings.Split(subject, ".")
	// 最少需要 7 段: domains, {domainID}, devices, {region}, {deviceType}, {deviceID}, down
	if len(parts) < 7 {
		return "", fmt.Errorf("invalid subject format: %s (expected at least 7 segments)", subject)
	}
	// deviceID 是倒数第二段
	deviceID := parts[len(parts)-2]
	if deviceID == "" || deviceID == "*" {
		return "", fmt.Errorf("deviceID is empty or wildcard in subject: %s", subject)
	}
	return deviceID, nil
}
