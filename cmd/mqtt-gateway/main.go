package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/datacenter/internal/gateway"
	"github.com/datacenter/internal/mqtt"
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

	// Publisher: NATS → Gateway 桥接
	natsPub := natspkg.NewPublisher(natsClient.JetStream())
	natsAdapter := gateway.NewNATSPublisher(natsPub)

	// MQTT 网关
	config := mqtt.Config{
		Port:          1883,
		MaxConnection: 100,
		KeepAlive:     60,
	}
	gw := mqtt.NewGateway(config, natsAdapter)

	if err := gw.Start(); err != nil {
		log.Fatalf("failed to start MQTT gateway: %v", err)
	}
	fmt.Println("MQTT Gateway listening on :1883")

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("MQTT Gateway shutting down...")
	gw.Stop()
}
