package mqtt

import (
	"fmt"
	"net"
	"sync"

	"github.com/datacenter/internal/gateway"
)

// Config MQTT 网关配置
type Config struct {
	Port          int    `yaml:"port"`
	MaxConnection int    `yaml:"max_connection"`
	KeepAlive     int    `yaml:"keep_alive"` // seconds, default 60
	TLSEnabled    bool   `yaml:"tls_enabled"`
	TLSCert       string `yaml:"tls_cert"`
	TLSKey        string `yaml:"tls_key"`
}

// DeviceStatusCallback 设备数据到达回调
type DeviceStatusCallback func(deviceID string)

// Gateway MQTT 协议网关
type Gateway struct {
	config    Config
	listener  net.Listener
	sessions  *SessionManager
	publisher gateway.Publisher
	codec     *Codec
	onData    DeviceStatusCallback
	quit      chan struct{}
	wg        sync.WaitGroup
	mu        sync.Mutex
	conns     map[net.Conn]struct{}  // 跟踪所有活跃连接
	clients   map[string]*Client     // deviceID → Client 映射，用于下行命令
}

// NewGateway 创建 MQTT 网关
func NewGateway(config Config, publisher gateway.Publisher) *Gateway {
	return &Gateway{
		config:    config,
		sessions:  NewSessionManager(),
		publisher: publisher,
		codec:     NewCodec(),
		quit:      make(chan struct{}),
		conns:     make(map[net.Conn]struct{}),
		clients:   make(map[string]*Client),
	}
}

// SetOnDataReceived 设置数据到达回调
func (g *Gateway) SetOnDataReceived(cb DeviceStatusCallback) {
	g.onData = cb
}

// Start 启动 MQTT 网关
func (g *Gateway) Start() error {
	addr := fmt.Sprintf(":%d", g.config.Port)
	var err error
	g.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	fmt.Println("[MQTT] listening on ", addr)

	g.wg.Add(1)
	go g.acceptLoop()
	return nil
}

// Stop 停止 MQTT 网关
func (g *Gateway) Stop() error {
	close(g.quit)
	if g.listener != nil {
		g.listener.Close()
	}
	// 关闭所有活跃连接，让 ReadLoop 退出
	g.mu.Lock()
	for conn := range g.conns {
		conn.Close()
	}
	g.conns = make(map[net.Conn]struct{})
	g.mu.Unlock()
	g.wg.Wait()
	fmt.Println("[MQTT] gateway stopped")
	return nil
}

// OnDeviceStatusChanged 设备状态变更
func (g *Gateway) OnDeviceStatusChanged(deviceID string, status gateway.DeviceStatus) {
	fmt.Println("[MQTT] device status changed:", deviceID)
}

// SendCommand 向已连接的 MQTT 设备发送下行命令
func (g *Gateway) SendCommand(deviceID string, payload []byte) error {
	g.mu.Lock()
	client, ok := g.clients[deviceID]
	g.mu.Unlock()

	if !ok {
		return fmt.Errorf("device %s not connected to MQTT gateway", deviceID)
	}

	// 使用设备订阅的主题作为下行通道
	// 默认使用 devices/{deviceID}/commands 主题
	topic := fmt.Sprintf("devices/%s/commands", deviceID)
	return client.SendCommand(topic, payload)
}

// GetConnectedDevices 返回当前已连接的 MQTT 设备 ID 列表
func (g *Gateway) GetConnectedDevices() []string {
	g.mu.Lock()
	defer g.mu.Unlock()

	devices := make([]string, 0, len(g.clients))
	for id := range g.clients {
		devices = append(devices, id)
	}
	return devices
}

// acceptLoop 接受连接
func (g *Gateway) acceptLoop() {
	defer g.wg.Done()
	fmt.Printf("[MQTT-GW] 🎧 等待连接...\n")
	for {
		conn, err := g.listener.Accept()
		if err != nil {
			select {
			case <-g.quit:
				fmt.Printf("[MQTT-GW] acceptLoop 退出\n")
				return
			default:
				fmt.Printf("[MQTT-GW] ❌ accept error: %v\n", err)
				continue
			}
		}
		fmt.Printf("[MQTT-GW] 🔗 收到新连接: %s\n", conn.RemoteAddr())
		g.wg.Add(1)
		go g.handleConnection(conn)
	}
}

// handleConnection 处理单个 MQTT 连接
func (g *Gateway) handleConnection(conn net.Conn) {
	defer g.wg.Done()
	defer func() {
		fmt.Printf("[MQTT-GW] 🔌 连接关闭: %s\n", conn.RemoteAddr())
		conn.Close()
		// 注销连接
		g.mu.Lock()
		delete(g.conns, conn)
		g.mu.Unlock()
	}()

	// 注册连接
	g.mu.Lock()
	g.conns[conn] = struct{}{}
	g.mu.Unlock()

	fmt.Printf("[MQTT-GW] 🔗 新连接: %s (当前连接数: %d)\n", conn.RemoteAddr(), len(g.conns))

	client := NewClient(conn, g.codec, g.sessions, g.publisher)
	client.onData = g.onData

	// 设置认证回调：设备认证成功后注册到 clients map
	client.onAuth = func(deviceID string) {
		g.mu.Lock()
		g.clients[deviceID] = client
		g.mu.Unlock()
		fmt.Printf("[MQTT-GW] 📋 设备已注册: %s (当前设备数: %d)\n", deviceID, len(g.clients))
	}

	if err := client.ReadLoop(); err != nil {
		fmt.Printf("[MQTT-GW] ❌ client error: %v\n", err)
	}

	// 连接结束时注销设备
	if client.clientID != "" {
		g.mu.Lock()
		delete(g.clients, client.clientID)
		g.mu.Unlock()
		fmt.Printf("[MQTT-GW] 📋 设备注销: %s (当前设备数: %d)\n", client.clientID, len(g.clients))
	}
}
