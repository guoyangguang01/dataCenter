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
	"github.com/datacenter/internal/model"
	"github.com/datacenter/internal/rule"
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
	); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// 服务层
	deviceService := device.NewService(db)
	domainService := domain.NewService(db)
	modelService := model.NewService(db)
	alertService := alert.NewAlertService(db)

	// 规则引擎
	registry := rule.NewRegistry()
	rule.RegisterBuiltinNodes(registry, nil) // nil publisher for now
	rule.RegisterScriptNode(registry)
	ruleEngine := rule.NewEngine(registry)

	ruleConfigService := rule.NewRuleConfigService(db, ruleEngine)
	if err := ruleConfigService.InitDB(); err != nil {
		log.Fatalf("failed to init rule config db: %v", err)
	}
	if err := ruleConfigService.LoadAll(); err != nil {
		log.Printf("warning: failed to load rules: %v", err)
	}

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

	// 启动
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced shutdown: %v", err)
	}
	fmt.Println("Server exited")
}
