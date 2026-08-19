package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/datacenter/internal/gateway"
	"github.com/datacenter/internal/modbus"
	natspkg "github.com/datacenter/pkg/nats"
)

func main() {
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

	// Modbus 网关
	config := modbus.Config{
		Port:         502,
		PollInterval: 10,
		SlaveIDs:     []int{1},
	}
	gw := modbus.NewGateway(config, natsAdapter)

	if err := gw.Start(); err != nil {
		log.Fatalf("failed to start Modbus gateway: %v", err)
	}
	fmt.Println("Modbus Gateway listening on :502")

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Modbus Gateway shutting down...")
	gw.Stop()
}
