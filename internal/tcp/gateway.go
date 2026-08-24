package tcp

import (
	"fmt"
	"net"
	"sync"

	"github.com/datacenter/internal/gateway"
)

type Config struct {
	Port          int    `yaml:"port"`
	MaxConnection int    `yaml:"max_connection"`
	Heartbeat     int    `yaml:"heartbeat"` // seconds
	TLSEnabled    bool   `yaml:"tls_enabled"`
	TLSCert       string `yaml:"tls_cert"`
	TLSKey        string `yaml:"tls_key"`
}

type DeviceStatusCallback func(deviceID string)

type Gateway struct {
	config    Config
	listener  net.Listener
	publisher gateway.Publisher
	onData    DeviceStatusCallback
	quit      chan struct{}
	wg        sync.WaitGroup
	mu        sync.Mutex
	conns     map[net.Conn]struct{} // 跟踪所有活跃连接
}

func NewGateway(config Config, publisher gateway.Publisher) *Gateway {
	return &Gateway{
		config:    config,
		publisher: publisher,
		quit:      make(chan struct{}),
		conns:     make(map[net.Conn]struct{}),
	}
}

func (g *Gateway) SetOnDataReceived(cb DeviceStatusCallback) {
	g.onData = cb
}

func (g *Gateway) Start() error {
	addr := fmt.Sprintf(":%d", g.config.Port)
	var err error
	g.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	fmt.Println("[TCP] listening on", addr)
	g.wg.Add(1)
	go g.acceptLoop()
	return nil
}

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
	fmt.Println("[TCP] gateway stopped")
	return nil
}

func (g *Gateway) OnDeviceStatusChanged(deviceID string, status gateway.DeviceStatus) {
	fmt.Println("[TCP] device status changed:", deviceID, status)
}

func (g *Gateway) acceptLoop() {
	defer g.wg.Done()
	fmt.Printf("[TCP-GW] 🎧 等待连接...\n")
	for {
		conn, err := g.listener.Accept()
		if err != nil {
			select {
			case <-g.quit:
				fmt.Printf("[TCP-GW] acceptLoop 退出\n")
				return
			default:
				fmt.Printf("[TCP-GW] ❌ accept error: %v\n", err)
				continue
			}
		}
		fmt.Printf("[TCP-GW] 🔗 收到新连接: %s\n", conn.RemoteAddr())
		g.wg.Add(1)
		go g.handleConnection(conn)
	}
}

func (g *Gateway) handleConnection(conn net.Conn) {
	defer g.wg.Done()
	defer func() {
		fmt.Printf("[TCP-GW] 🔌 连接关闭: %s\n", conn.RemoteAddr())
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

	fmt.Printf("[TCP-GW] 🔗 新连接: %s (当前连接数: %d)\n", conn.RemoteAddr(), len(g.conns))

	client := NewClient(conn, g.publisher)
	client.onData = g.onData
	if err := client.ReadLoop(); err != nil {
		fmt.Printf("[TCP-GW] ❌ client error: %v\n", err)
	}
}
