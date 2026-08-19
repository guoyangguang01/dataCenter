package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/datacenter/internal/alert"
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

	fmt.Println("Alert Service started")
	fmt.Printf("  Database: alert-service.db\n")

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Alert Service shutting down...")
}
