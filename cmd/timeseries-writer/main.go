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
	// TDengine 写入器 (原生驱动)
	writerConfig := timeseries.Config{
		DSN:            "root:taosdata@http(localhost:6041)/",
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

	// 创建持久消费者并持续监听（重启后从上次位置继续）
	cons, err := natsClient.JetStream().CreateOrUpdateConsumer(ctx, "DEVICE_DATA", jetstream.ConsumerConfig{
		Durable:       "timeseries-writer-v3",
		FilterSubject: "domains.*.devices.*.*.*.up",
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverNewPolicy,
	})
	if err != nil {
		log.Fatalf("failed to create consumer: %v", err)
	}

	iter, err := cons.Messages()
	if err != nil {
		log.Fatalf("failed to get messages: %v", err)
	}

	go func() {
		msgCount := 0
		for {
			msg, err := iter.Next()
			if err != nil {
				log.Printf("[Timeseries] ❌ iterator error: %v", err)
				continue
			}

			msgCount++
			fmt.Printf("[Timeseries] 📨 收到消息 #%d: subject=%s len=%d\n",
				msgCount, msg.Subject(), len(msg.Data()))

			var env message.DeviceEnvelope
			if err := json.Unmarshal(msg.Data(), &env); err != nil {
				log.Printf("[Timeseries] ❌ unmarshal error: %v", err)
				msg.Ack()
				continue
			}

			fmt.Printf("[Timeseries] 📦 解析成功: device=%s domain=%s units=%d\n",
				env.DeviceID, env.DomainID, len(env.Units))

			if err := writer.Write(&env); err != nil {
				log.Printf("[Timeseries] ❌ write error: %v", err)
			} else {
				fmt.Printf("[Timeseries] ✅ 写入成功: device=%s\n", env.DeviceID)
			}
			msg.Ack()
		}
	}()

	fmt.Println("Timeseries Writer started")
	fmt.Printf("  TDengine DSN: %s\n", writerConfig.DSN)
	fmt.Printf("  Subscribed: domains.*.devices.*.*.*.up\n")

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Timeseries Writer shutting down...")
	iter.Stop()
}
