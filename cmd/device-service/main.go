package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/datacenter/internal/device"
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

	_ = statusManager
	_ = authManager

	fmt.Println("Device Service started")
	fmt.Printf("  Database: device-service.db\n")
	fmt.Printf("  Redis: localhost:6379\n")

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Device Service shutting down...")
}
