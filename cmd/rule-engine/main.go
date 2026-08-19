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
	"github.com/datacenter/internal/rule"
	natspkg "github.com/datacenter/pkg/nats"
	"github.com/nats-io/nats.go/jetstream"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// 数据库
	db, err := gorm.Open(sqlite.Open("rule-engine.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	// 规则引擎
	registry := rule.NewRegistry()
	rule.RegisterBuiltinNodes(registry, nil)
	rule.RegisterScriptNode(registry)
	ruleEngine := rule.NewEngine(registry)

	// 规则持久化服务
	ruleConfigService := rule.NewRuleConfigService(db, ruleEngine)
	if err := ruleConfigService.InitDB(); err != nil {
		log.Fatalf("failed to init rule config db: %v", err)
	}
	if err := ruleConfigService.LoadAll(); err != nil {
		log.Printf("warning: failed to load rules: %v", err)
	}

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
		FilterSubject: "domains.>.devices.>.>.>.up",
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		log.Fatalf("failed to create consumer: %v", err)
	}

	// 持续消费消息
	iter, err := cons.Messages()
	if err != nil {
		log.Fatalf("failed to get messages: %v", err)
	}

	go func() {
		for {
			msg, err := iter.Next()
			if err != nil {
				log.Printf("[RuleEngine] iterator error: %v", err)
				continue
			}

			var env message.DeviceEnvelope
			if err := json.Unmarshal(msg.Data(), &env); err != nil {
				log.Printf("[RuleEngine] unmarshal error: %v", err)
				msg.Ack()
				continue
			}

			if err := ruleEngine.Process(ctx, &env); err != nil {
				log.Printf("[RuleEngine] process error: %v", err)
			}
			msg.Ack()
		}
	}()

	fmt.Println("Rule Engine started")
	fmt.Printf("  Database: rule-engine.db\n")
	fmt.Printf("  Subscribed: domains.>.devices.>.>.>.up\n")

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Rule Engine shutting down...")
	iter.Stop()
}
