package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	v1 "github.com/datacenter/api/v1"
	"github.com/datacenter/internal/device"
	"github.com/datacenter/internal/gateway"
	"github.com/datacenter/internal/modbus"
	"github.com/datacenter/internal/mqtt"
	"github.com/datacenter/internal/tcp"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// 数据库
	db, err := gorm.Open(sqlite.Open("device-service.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	if err := db.AutoMigrate(&device.Device{}); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// Redis
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rdb.Close()

	// 服务层
	deviceService := device.NewService(db)
	statusManager := device.NewStatusManager(rdb)
	authManager := device.NewAuthManager(rdb, deviceService)

	// NATS 连接（不可用时降级为日志 publisher）
	var gwPublisher gateway.Publisher
	nc, natsErr := nats.Connect("nats://localhost:4222", nats.Name("device-service"))
	if natsErr != nil {
		fmt.Println("[NATS] not available, using log publisher:", natsErr)
		gwPublisher = gateway.NewLogPublisher()
	} else {
		gwPublisher = &gateway.SimpleNATSPublisher{Conn: nc}
		fmt.Println("[NATS] connected to nats://localhost:4222")
	}
	defer func() {
		if nc != nil {
			nc.Close()
		}
	}()

	// 网关启动器
	gatewayService := gateway.NewGatewayService(db)
	launcher := gateway.NewLauncher(gatewayService, gwPublisher)

	// 注册网关工厂：收到数据时更新设备在线状态
	launcher.Register(gateway.TypeMQTT, func(configStr string, pub gateway.Publisher) (gateway.GatewayAdapter, error) {
		cfg, err := gateway.ParseMQTTConfig(configStr)
		if err != nil {
			return nil, err
		}
		gw := mqtt.NewGateway(mqtt.Config{
			Port:          cfg.Port,
			MaxConnection: cfg.MaxConnection,
			KeepAlive:     cfg.KeepAlive,
		}, pub)
		gw.SetOnDataReceived(func(deviceID string) {
			ctx := context.Background()
			if err := statusManager.UpdateOnline(ctx, deviceID, true); err != nil {
				log.Printf("[Status] failed to update online: %v", err)
			}
			db.Model(&device.Device{}).Where("id = ?", deviceID).Updates(map[string]interface{}{
				"online":    true,
				"last_seen": time.Now(),
			})
		})
		return gw, nil
	})

	launcher.Register(gateway.TypeTCP, func(configStr string, pub gateway.Publisher) (gateway.GatewayAdapter, error) {
		cfg, err := gateway.ParseTCPConfig(configStr)
		if err != nil {
			return nil, err
		}
		gw := tcp.NewGateway(tcp.Config{
			Port:          cfg.Port,
			MaxConnection: cfg.MaxConnection,
			Heartbeat:     cfg.Heartbeat,
		}, pub)
		gw.SetOnDataReceived(func(deviceID string) {
			ctx := context.Background()
			if err := statusManager.UpdateOnline(ctx, deviceID, true); err != nil {
				log.Printf("[Status] failed to update online: %v", err)
			}
			db.Model(&device.Device{}).Where("id = ?", deviceID).Updates(map[string]interface{}{
				"online":    true,
				"last_seen": time.Now(),
			})
		})
		return gw, nil
	})

	launcher.Register(gateway.TypeModbus, func(configStr string, pub gateway.Publisher) (gateway.GatewayAdapter, error) {
		cfg, err := gateway.ParseModbusConfig(configStr)
		if err != nil {
			return nil, err
		}
		gw := modbus.NewGateway(modbus.Config{
			Port:         cfg.Port,
			PollInterval: cfg.PollInterval,
			SlaveIDs:     cfg.SlaveIDs,
		}, pub)
		gw.SetOnDataReceived(func(deviceID string) {
			ctx := context.Background()
			if err := statusManager.UpdateOnline(ctx, deviceID, true); err != nil {
				log.Printf("[Status] failed to update online: %v", err)
			}
			db.Model(&device.Device{}).Where("id = ?", deviceID).Updates(map[string]interface{}{
				"online":    true,
				"last_seen": time.Now(),
			})
		})
		return gw, nil
	})

	// 定时扫描离线设备（每 60 秒）
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			ctx := context.Background()
			offlineDevices, err := statusManager.ScanOffline(ctx)
			if err != nil {
				log.Printf("[Status] scan offline error: %v", err)
				continue
			}
			for _, deviceID := range offlineDevices {
				db.Model(&device.Device{}).Where("id = ?", deviceID).Update("online", false)
				log.Printf("[Status] device marked offline: %s", deviceID)
			}
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
	v1.NewDeviceHandler(deviceService).RegisterRoutes(api)

	// 设备认证接口
	api.POST("/devices/:id/authenticate", func(c *gin.Context) {
		deviceID := c.Param("id")
		var req struct {
			Credential string `json:"credential" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		dev, err := authManager.Authenticate(c.Request.Context(), deviceID, req.Credential)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"authenticated": true, "device": dev})
	})

	// 设备状态查询接口
	api.GET("/devices/:id/status", func(c *gin.Context) {
		deviceID := c.Param("id")
		status, err := statusManager.GetStatus(c.Request.Context(), deviceID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "status not found"})
			return
		}
		c.JSON(http.StatusOK, status)
	})

	srv := &http.Server{
		Addr:         ":8081",
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		fmt.Println("Device Service HTTP listening on :8081")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	fmt.Println("Device Service started")
	fmt.Printf("  Database: device-service.db\n")
	fmt.Printf("  Redis: localhost:6379\n")
	fmt.Printf("  HTTP: :8081\n")

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Device Service shutting down...")

	launcher.StopAll()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced shutdown: %v", err)
	}
	fmt.Println("Device Service stopped")
}
