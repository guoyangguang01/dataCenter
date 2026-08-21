package modbus

import (
	"fmt"
	"net"
	"sync"

	"github.com/datacenter/internal/gateway"
)

type Config struct {
	Port         int    `yaml:"port"`
	PollInterval int    `yaml:"poll_interval"` // seconds
	SlaveIDs     []int  `yaml:"slave_ids"`
}

type DeviceStatusCallback func(deviceID string)

type Gateway struct {
	config    Config
	listener  net.Listener
	publisher gateway.Publisher
	onData    DeviceStatusCallback
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
	fmt.Println("[Modbus] listening on", addr)
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
	fmt.Println("[Modbus] gateway stopped")
	return nil
}

func (g *Gateway) OnDeviceStatusChanged(deviceID string, status gateway.DeviceStatus) {
	fmt.Println("[Modbus] device status changed:", deviceID, status)
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
				fmt.Println("[Modbus] accept error:", err)
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
	client := NewClient(conn, g.publisher, g.config)
	client.onData = g.onData
	client.ReadLoop()
}
