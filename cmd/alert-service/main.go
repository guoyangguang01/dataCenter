package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	v1 "github.com/datacenter/api/v1"
	"github.com/datacenter/internal/alert"
	natspkg "github.com/datacenter/pkg/nats"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go/jetstream"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// 数据库
	db, err := gorm.Open(sqlite.Open("alert-service.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	// 告警服务
	alertService := alert.NewAlertService(db)
	if err := alertService.InitDB(); err != nil {
		log.Fatalf("failed to init alert db: %v", err)
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

	// 订阅 SYSTEM_EVENTS stream 中的告警事件（持久消费者）
	cons, err := natsClient.JetStream().CreateOrUpdateConsumer(ctx, "SYSTEM_EVENTS", jetstream.ConsumerConfig{
		Durable:       "alert-service",
		FilterSubject: "system.alerts.>",
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		log.Fatalf("failed to create alert consumer: %v", err)
	}

	iter, err := cons.Messages()
	if err != nil {
		log.Fatalf("failed to get messages iterator: %v", err)
	}

	// 持续消费告警事件
	go func() {
		for {
			msg, err := iter.Next()
			if err != nil {
				log.Printf("[Alert] iterator error: %v", err)
				continue
			}

			var event alert.AlertEvent
			if err := json.Unmarshal(msg.Data(), &event); err != nil {
				log.Printf("[Alert] unmarshal error: %v", err)
				msg.Ack()
				continue
			}

			fmt.Printf("[Alert] received: %s level=%s device=%s\n", event.AlertID, event.Level, event.DeviceID)

			if err := alertService.ProcessAlertEvent(&event); err != nil {
				log.Printf("[Alert] process error: %v", err)
			}
			msg.Ack()
		}
	}()

	// Gin HTTP 服务
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	api := r.Group("/api/v1")
	v1.NewAlertHandler(alertService).RegisterRoutes(api)

	// 告警统计接口
	api.GET("/alerts/stats", func(c *gin.Context) {
		count, err := alertService.CountRecent()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"count_24h": count})
	})

	srv := &http.Server{
		Addr:         ":8082",
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		fmt.Println("Alert Service HTTP listening on :8082")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	fmt.Println("Alert Service started")
	fmt.Printf("  Database: alert-service.db\n")
	fmt.Printf("  HTTP: :8082\n")
	fmt.Printf("  Subscribed: system.alerts.>\n")

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Alert Service shutting down...")

	iter.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced shutdown: %v", err)
	}
	fmt.Println("Alert Service stopped")
}
