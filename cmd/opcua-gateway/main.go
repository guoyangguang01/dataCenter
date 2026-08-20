package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/datacenter/internal/gateway"
	"github.com/datacenter/internal/opcua"
	natspkg "github.com/datacenter/pkg/nats"
)

func main() {
	// OPC UA 配置
	config := opcua.Config{
		Endpoint:     "opc.tcp://localhost:4840",
		PollInterval: 5,
		NodeIDs:      []string{"ns=2;s=Temperature", "ns=2;s=Humidity"},
		DeviceID:     "opcua-plc-001",
		DomainID:     "default",
	}

	// NATS 连接
	natsClient, err := natspkg.New(natspkg.DefaultConfig())
	if err != nil {
		log.Fatalf("failed to connect to NATS: %v", err)
	}
	defer natsClient.Close()

	// 确保 Stream 存在
	ctx := context.Background()
	if err := natspkg.EnsureAllStreams(ctx, natsClient.JetStream()); err != nil {
		log.Fatalf("failed to ensure streams: %v", err)
	}

	// Publisher
	natsPub := natspkg.NewPublisher(natsClient.JetStream())
	natsAdapter := gateway.NewNATSPublisher(natsPub)

	// OPC UA 网关
	gw := opcua.NewGateway(config, natsAdapter)

	if err := gw.Start(); err != nil {
		log.Fatalf("failed to start OPC UA gateway: %v", err)
	}

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("OPC UA Gateway shutting down...")
	gw.Stop()
}
