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

type Gateway struct {
	config    Config
	listener  net.Listener
	publisher gateway.Publisher
	quit      chan struct{}
	wg        sync.WaitGroup
}

func NewGateway(config Config, publisher gateway.Publisher) *Gateway {
	return &Gateway{
		config:    config,
		publisher: publisher,
		quit:      make(chan struct{}),
	}
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
	g.wg.Wait()
	fmt.Println("[TCP] gateway stopped")
	return nil
}

func (g *Gateway) OnDeviceStatusChanged(deviceID string, status gateway.DeviceStatus) {
	fmt.Println("[TCP] device status changed:", deviceID, status)
}

func (g *Gateway) acceptLoop() {
	defer g.wg.Done()
	for {
		conn, err := g.listener.Accept()
		if err != nil {
			select {
			case <-g.quit:
				return
			default:
				fmt.Println("[TCP] accept error:", err)
				continue
			}
		}
		g.wg.Add(1)
		go g.handleConnection(conn)
	}
}

func (g *Gateway) handleConnection(conn net.Conn) {
	defer g.wg.Done()
	defer conn.Close()
	client := NewClient(conn, g.publisher)
	if err := client.ReadLoop(); err != nil {
		fmt.Println("[TCP] client error:", err)
	}
}
