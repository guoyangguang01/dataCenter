package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/datacenter/internal/message"
	"github.com/datacenter/pkg/nats"
	nats_go "github.com/nats-io/nats.go"
)

// DeviceStatus 设备在线状态
type DeviceStatus int

const (
	DeviceOffline DeviceStatus = 0
	DeviceOnline  DeviceStatus = 1
)

// GatewayAdapter 网关适配器接口
// 每个协议网关必须实现此接口
type GatewayAdapter interface {
	// Start 启动网关（监听端口、初始化连接池等）
	Start() error

	// Stop 停止网关
	Stop() error

	// OnDeviceStatusChanged 设备状态变更通知
	OnDeviceStatusChanged(deviceID string, status DeviceStatus)
}

// Publisher 消息发布器接口
type Publisher interface {
	PublishEnvelope(env *message.DeviceEnvelope) error
}

// Config 网关基础配置
type Config struct {
	Protocol      string `yaml:"protocol"`
	Port          int    `yaml:"port"`
	MaxConnection int    `yaml:"max_connection"`
}

// Registry 网关注册中心
type Registry struct {
	adapters map[string]GatewayAdapter
	publisher Publisher
}

// NewRegistry 创建网关注册中心
func NewRegistry(publisher Publisher) *Registry {
	return &Registry{
		adapters: make(map[string]GatewayAdapter),
		publisher: publisher,
	}
}

// Register 注册网关适配器
func (r *Registry) Register(protocol string, adapter GatewayAdapter) {
	r.adapters[protocol] = adapter
}

// Get 获取网关适配器
func (r *Registry) Get(protocol string) (GatewayAdapter, bool) {
	adapter, ok := r.adapters[protocol]
	return adapter, ok
}

// StartAll 启动所有已注册的网关
func (r *Registry) StartAll() error {
	for protocol, adapter := range r.adapters {
		if err := adapter.Start(); err != nil {
			return fmt.Errorf("failed to start %s gateway: %w", protocol, err)
		}
	}
	return nil
}

// StopAll 停止所有网关
func (r *Registry) StopAll() {
	for _, adapter := range r.adapters {
		adapter.Stop()
	}
}

// NATSPublisher 基于 NATS 的消息发布器实现
type NATSPublisher struct {
	pub *nats.Publisher
}

// NewNATSPublisher 创建 NATS 发布器
func NewNATSPublisher(pub *nats.Publisher) *NATSPublisher {
	return &NATSPublisher{pub: pub}
}

// PublishEnvelope 发布 DeviceEnvelope
func (p *NATSPublisher) PublishEnvelope(env *message.DeviceEnvelope) error {
	subject := nats.DeviceReportTopic(
		env.DomainID,
		env.Metadata["region"],
		env.Metadata["device_type"],
		env.DeviceID,
	)
	return p.pub.PublishJSON(context.Background(), subject, env)
}

// SimpleNATSPublisher 使用原生 NATS 连接的简单发布器
type SimpleNATSPublisher struct {
	Conn *nats_go.Conn
}

func (p *SimpleNATSPublisher) PublishEnvelope(env *message.DeviceEnvelope) error {
	region := "default"
	deviceType := "unknown"
	if env.Metadata != nil {
		if v, ok := env.Metadata["region"]; ok && v != "" {
			region = v
		}
		if v, ok := env.Metadata["device_type"]; ok && v != "" {
			deviceType = v
		}
	}
	subject := nats.DeviceReportTopic(env.DomainID, region, deviceType, env.DeviceID)
	data, err := json.Marshal(env)
	if err != nil {
		fmt.Printf("[NATS-Pub] ❌ marshal error: device=%s err=%v\n", env.DeviceID, err)
		return err
	}

	fmt.Printf("[NATS-Pub] 📤 发布到 NATS: subject=%s device=%s units=%d payload=%d bytes\n",
		subject, env.DeviceID, len(env.Units), len(data))

	if err := p.Conn.Publish(subject, data); err != nil {
		fmt.Printf("[NATS-Pub] ❌ 发布失败: subject=%s err=%v\n", subject, err)
		return err
	}

	// Flush 确保消息真正发送出去
	if err := p.Conn.Flush(); err != nil {
		fmt.Printf("[NATS-Pub] ⚠️ Flush 失败: %v\n", err)
	}

	fmt.Printf("[NATS-Pub] ✅ 发布成功: subject=%s (%d bytes)\n", subject, len(data))
	return nil
}
