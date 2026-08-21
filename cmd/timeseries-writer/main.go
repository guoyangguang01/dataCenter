package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/datacenter/internal/message"
	"github.com/datacenter/internal/timeseries"
	natspkg "github.com/datacenter/pkg/nats"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	// TDengine 写入器
	writerConfig := timeseries.Config{
		RESTAddr:       "http://localhost:6041",
		BatchSize:      100,
		FlushInterval:  5,
		BufferCapacity: 50000,
	}
	writer := timeseries.NewWriter(writerConfig)
	if err := writer.Start(); err != nil {
		log.Printf("warning: failed to start TDengine writer: %v (TDengine may not be running)", err)
	}
	defer writer.Stop()

	// NATS 连接
	natsClient, err := natspkg.New(natspkg.DefaultConfig())
	if err != nil {
		log.Fatalf("failed to connect to NATS: %v", err)
	}
	defer natsClient.Close()

	ctx := context.Background()
	if err := natspkg.EnsureAllStreams(ctx, natsClient.JetStream()); err != nil {
		log.Fatalf("failed to ensure streams: %v", err)
	}

	// 创建消费者并持续监听
	cons, err := natsClient.JetStream().CreateOrUpdateConsumer(ctx, "DEVICE_DATA", jetstream.ConsumerConfig{
		FilterSubject: "domains.*.devices.*.*.*.up",
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		log.Fatalf("failed to create consumer: %v", err)
	}

	iter, err := cons.Messages()
	if err != nil {
		log.Fatalf("failed to get messages: %v", err)
	}

	go func() {
		for {
			msg, err := iter.Next()
			if err != nil {
				log.Printf("[Timeseries] iterator error: %v", err)
				continue
			}

			fmt.Printf("[Timeseries] received message: subject=%s len=%d\n", msg.Subject(), len(msg.Data()))

			var env message.DeviceEnvelope
			if err := json.Unmarshal(msg.Data(), &env); err != nil {
				log.Printf("[Timeseries] unmarshal error: %v", err)
				msg.Ack()
				continue
			}

			if err := writer.Write(&env); err != nil {
				log.Printf("[Timeseries] write error: %v", err)
			} else {
				fmt.Printf("[Timeseries] wrote data for device=%s\n", env.DeviceID)
			}
			msg.Ack()
		}
	}()

	fmt.Println("Timeseries Writer started")
	fmt.Printf("  TDengine REST: %s\n", writerConfig.RESTAddr)
	fmt.Printf("  Subscribed: domains.*.devices.*.*.*.up\n")

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Timeseries Writer shutting down...")
	iter.Stop()
}
