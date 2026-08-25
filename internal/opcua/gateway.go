package opcua

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/datacenter/internal/gateway"
	"github.com/datacenter/internal/message"
)

// Config OPC UA 网关配置
type Config struct {
	Endpoint     string   `yaml:"endpoint"`      // OPC UA 服务端地址，如 opc.tcp://192.168.1.100:4840
	PollInterval int      `yaml:"poll_interval"` // 轮询间隔（秒）
	NodeIDs      []string `yaml:"node_ids"`      // 要读取的 Node ID 列表
	DeviceID     string   `yaml:"device_id"`     // 设备标识
	DomainID     string   `yaml:"domain_id"`     // 业务域
}

// Gateway OPC UA 协议网关
type Gateway struct {
	config    Config
	client    *Client
	publisher gateway.Publisher
	quit      chan struct{}
	wg        sync.WaitGroup
}

// NewGateway 创建 OPC UA 网关
func NewGateway(config Config, publisher gateway.Publisher) *Gateway {
	return &Gateway{
		config:    config,
		publisher: publisher,
		quit:      make(chan struct{}),
	}
}

// Start 启动 OPC UA 网关
func (g *Gateway) Start() error {
	// 连接 OPC UA Server
	client, err := Connect(g.config.Endpoint)
	if err != nil {
		return fmt.Errorf("failed to start OPC UA gateway: %w", err)
	}
	g.client = client

	// 启动轮询协程
	g.wg.Add(1)
	go g.pollLoop()

	fmt.Printf("[OPC UA] gateway started, polling %d nodes every %ds\n", len(g.config.NodeIDs), g.config.PollInterval)
	return nil
}

// Stop 停止 OPC UA 网关
func (g *Gateway) Stop() error {
	close(g.quit)
	g.wg.Wait()
	if g.client != nil {
		g.client.Close()
	}
	fmt.Println("[OPC UA] gateway stopped")
	return nil
}

// OnDeviceStatusChanged 设备状态变更
func (g *Gateway) OnDeviceStatusChanged(deviceID string, status gateway.DeviceStatus) {
	fmt.Println("[OPC UA] device status changed:", deviceID, status)
}

// SendCommand 向 OPC UA 节点写入值
// payload 格式: JSON {"node_id": "ns=2;s=Temperature", "value": 25.0}
func (g *Gateway) SendCommand(deviceID string, payload []byte) error {
	if g.client == nil {
		return fmt.Errorf("OPC UA client not connected")
	}

	// 校验 deviceID 是否匹配
	if deviceID != g.config.DeviceID {
		return fmt.Errorf("device %s not associated with this OPC UA gateway (expected %s)", deviceID, g.config.DeviceID)
	}

	// 解析写入请求
	var req WriteRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return fmt.Errorf("invalid write request payload: %w", err)
	}

	if req.NodeID == "" {
		return fmt.Errorf("node_id is required in write request")
	}

	return g.client.WriteNode(req.NodeID, req.Value)
}

// GetConnectedDevices OPC UA 网关返回配置的设备 ID
func (g *Gateway) GetConnectedDevices() []string {
	if g.config.DeviceID == "" {
		return nil
	}
	return []string{g.config.DeviceID}
}

// pollLoop 定时轮询 OPC UA 节点
func (g *Gateway) pollLoop() {
	defer g.wg.Done()
	ticker := time.NewTicker(time.Duration(g.config.PollInterval) * time.Second)
	defer ticker.Stop()

	// 立即执行一次
	g.poll()

	for {
		select {
		case <-g.quit:
			return
		case <-ticker.C:
			g.poll()
		}
	}
}

// poll 读取所有节点并发布到 NATS
func (g *Gateway) poll() {
	values, err := g.client.ReadNodes(g.config.NodeIDs)
	if err != nil {
		fmt.Printf("[OPC UA] poll error: %v\n", err)
		return
	}

	if len(values) == 0 {
		return
	}

	// 构建 DeviceEnvelope
	env := message.NewDeviceEnvelope(
		g.config.DeviceID,
		g.config.DomainID,
		"opcua_device",
		message.DataType,
	)
	env.Metadata["protocol"] = "opcua"
	env.Metadata["endpoint"] = g.config.Endpoint

	// 将所有节点值序列化为 JSON
	payload, err := json.Marshal(values)
	if err != nil {
		fmt.Printf("[OPC UA] marshal error: %v\n", err)
		return
	}

	env.AddUnit("opcua_nodes", payload)

	// 发布到 NATS
	if err := g.publisher.PublishEnvelope(env); err != nil {
		fmt.Printf("[OPC UA] publish error: %v\n", err)
	}
}
