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
	"github.com/datacenter/internal/alert"
	"github.com/datacenter/internal/device"
	"github.com/datacenter/internal/domain"
	"github.com/datacenter/internal/gateway"
	"github.com/datacenter/internal/modbus"
	"github.com/datacenter/internal/mqtt"
	"github.com/datacenter/internal/opcua"
	"github.com/datacenter/internal/model"
	"github.com/datacenter/internal/rule"
	"github.com/datacenter/internal/tcp"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// 数据库
	db, err := gorm.Open(sqlite.Open("datacenter.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	// AutoMigrate 所有表
	if err := db.AutoMigrate(
		&device.Device{},
		&domain.Domain{},
		&domain.DomainMember{},
		&model.ThingModel{},
		&model.DeviceModelBinding{},
		&rule.RuleConfig{},
		&alert.WebhookConfig{},
		&alert.AlertLog{},
		&gateway.GatewayConfig{},
	); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// 服务层
	deviceService := device.NewService(db)
	domainService := domain.NewService(db)
	modelService := model.NewService(db)
	alertService := alert.NewAlertService(db)
	gatewayService := gateway.NewGatewayService(db)

	// 规则引擎
	registry := rule.NewRegistry()
	rule.RegisterBuiltinNodes(registry, nil)
	rule.RegisterScriptNode(registry)
	ruleEngine := rule.NewEngine(registry)

	ruleConfigService := rule.NewRuleConfigService(db, ruleEngine)
	if err := ruleConfigService.InitDB(); err != nil {
		log.Fatalf("failed to init rule config db: %v", err)
	}
	if err := ruleConfigService.LoadAll(); err != nil {
		log.Printf("warning: failed to load rules: %v", err)
	}

	// 网关启动器（使用 console 的 NATS publisher）
	// 注意：这里创建一个简单的 publisher 用于网关管理
	// 实际的 NATS 连接在各网关启动时单独建立
	launcher := gateway.NewLauncher(gatewayService, nil)

	// 注册网关工厂
	launcher.Register(gateway.TypeMQTT, func(configStr string, pub gateway.Publisher) (gateway.GatewayAdapter, error) {
		cfg, err := gateway.ParseMQTTConfig(configStr)
		if err != nil {
			return nil, err
		}
		return mqtt.NewGateway(mqtt.Config{
			Port:          cfg.Port,
			MaxConnection: cfg.MaxConnection,
			KeepAlive:     cfg.KeepAlive,
		}, pub), nil
	})

	launcher.Register(gateway.TypeTCP, func(configStr string, pub gateway.Publisher) (gateway.GatewayAdapter, error) {
		cfg, err := gateway.ParseTCPConfig(configStr)
		if err != nil {
			return nil, err
		}
		return tcp.NewGateway(tcp.Config{
			Port:          cfg.Port,
			MaxConnection: cfg.MaxConnection,
			Heartbeat:     cfg.Heartbeat,
		}, pub), nil
	})

	launcher.Register(gateway.TypeModbus, func(configStr string, pub gateway.Publisher) (gateway.GatewayAdapter, error) {
		cfg, err := gateway.ParseModbusConfig(configStr)
		if err != nil {
			return nil, err
		}
		return modbus.NewGateway(modbus.Config{
			Port:         cfg.Port,
			PollInterval: cfg.PollInterval,
			SlaveIDs:     cfg.SlaveIDs,
		}, pub), nil
	})

	launcher.Register(gateway.TypeOPCUA, func(configStr string, pub gateway.Publisher) (gateway.GatewayAdapter, error) {
		cfg, err := gateway.ParseOPCUAConfig(configStr)
		if err != nil {
			return nil, err
		}
		return opcua.NewGateway(opcua.Config{
			Endpoint:     cfg.Endpoint,
			PollInterval: cfg.PollInterval,
			NodeIDs:      cfg.NodeIDs,
			DeviceID:     cfg.DeviceID,
			DomainID:     cfg.DomainID,
		}, pub), nil
	})

	// Gin
	r := gin.Default()

	// CORS
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

	// 路由
	api := r.Group("/api/v1")
	v1.NewDeviceHandler(deviceService).RegisterRoutes(api)
	v1.NewDomainHandler(domainService).RegisterRoutes(api)
	v1.NewModelHandler(modelService).RegisterRoutes(api)
	v1.NewRuleHandler(ruleConfigService).RegisterRoutes(api)
	v1.NewAlertHandler(alertService).RegisterRoutes(api)
	v1.NewGatewayHandler(gatewayService, launcher).RegisterRoutes(api)

	// 启动 HTTP 服务
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		fmt.Println("Console server listening on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Shutting down...")

	// 停止所有网关
	launcher.StopAll()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced shutdown: %v", err)
	}
	fmt.Println("Server exited")
}
