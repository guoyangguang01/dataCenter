package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/datacenter/internal/gateway"
	"github.com/datacenter/internal/tcp"
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

	// TCP 网关
	config := tcp.Config{
		Port:          9000,
		MaxConnection: 100,
		Heartbeat:     30,
	}
	gw := tcp.NewGateway(config, natsAdapter)

	if err := gw.Start(); err != nil {
		log.Fatalf("failed to start TCP gateway: %v", err)
	}
	fmt.Println("TCP Gateway listening on :9000")

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("TCP Gateway shutting down...")
	gw.Stop()
}
