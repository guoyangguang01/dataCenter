package alert

import (
	"time"
)

// AlertLevel 告警级别
type AlertLevel string

const (
	LevelCritical AlertLevel = "critical"
	LevelWarning  AlertLevel = "warning"
	LevelInfo     AlertLevel = "info"
)

// AlertEvent 告警事件
type AlertEvent struct {
	AlertID   string            `json:"alert_id"`
	DomainID  string            `json:"domain_id"`
	DeviceID  string            `json:"device_id"`
	RuleID    string            `json:"rule_id"`
	Level     AlertLevel        `json:"level"`
	Title     string            `json:"title"`
	Message   string            `json:"message"`
	Timestamp int64             `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// WebhookConfig Webhook 配置
type WebhookConfig struct {
	ID         string            `json:"id" gorm:"primaryKey;size:64"`
	Name       string            `json:"name" gorm:"size:128"`
	DomainID   string            `json:"domain_id" gorm:"size:64;index"`
	URL        string            `json:"url" gorm:"size:512"`
	Method     string            `json:"method" gorm:"size:8;default:POST"`
	Headers    string            `json:"headers" gorm:"type:text"` // JSON
	Filter     WebhookFilter     `json:"filter" gorm:"serializer:json"`
	RateLimit  RateLimitConfig   `json:"rate_limit" gorm:"serializer:json"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// WebhookFilter 告警过滤条件
type WebhookFilter struct {
	Levels      []AlertLevel `json:"levels"`
	DeviceTypes []string     `json:"device_types"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	MaxPerMinute int `json:"max_per_minute"`
	DedupWindow  int `json:"dedup_window"` // seconds
}

// AlertLog 告警日志
type AlertLog struct {
	ID         int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	AlertID    string     `json:"alert_id" gorm:"index"`
	WebhookID  string     `json:"webhook_id"`
	Status     string     `json:"status"` // sent / failed / rate_limited
	Response   string     `json:"response"`
	CreatedAt  time.Time  `json:"created_at"`
}

// CreateWebhookRequest 创建 Webhook 请求
type CreateWebhookRequest struct {
	Name     string          `json:"name" binding:"required"`
	DomainID string          `json:"domain_id" binding:"required"`
	URL      string          `json:"url" binding:"required"`
	Method   string          `json:"method"`
	Headers  map[string]string `json:"headers"`
	Filter   WebhookFilter   `json:"filter"`
	RateLimit RateLimitConfig `json:"rate_limit"`
}
