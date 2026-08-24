package alert

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// AlertService 告警管理服务
type AlertService struct {
	db     *gorm.DB
	sender *WebhookSender
}

// NewAlertService 创建告警管理服务
func NewAlertService(db *gorm.DB) *AlertService {
	return &AlertService{
		db:     db,
		sender: NewWebhookSender(),
	}
}

// InitDB 初始化数据库表
func (s *AlertService) InitDB() error {
	return s.db.AutoMigrate(&WebhookConfig{}, &AlertLog{})
}

// CreateWebhook 创建 Webhook 配置
func (s *AlertService) CreateWebhook(req CreateWebhookRequest) (*WebhookConfig, error) {
	headersJSON := "{}"
	if req.Headers != nil {
		data, err := json.Marshal(req.Headers)
		if err != nil {
			return nil, err
		}
		headersJSON = string(data)
	}

	config := &WebhookConfig{
		ID:        fmt.Sprintf("wh_%d", time.Now().UnixNano()),
		Name:      req.Name,
		DomainID:  req.DomainID,
		URL:       req.URL,
		Method:    req.Method,
		Headers:   headersJSON,
		Filter:    req.Filter,
		RateLimit: req.RateLimit,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if config.Method == "" {
		config.Method = "POST"
	}

	if err := s.db.Create(config).Error; err != nil {
		return nil, err
	}

	return config, nil
}

// GetWebhook 获取 Webhook 配置
func (s *AlertService) GetWebhook(id string) (*WebhookConfig, error) {
	var config WebhookConfig
	if err := s.db.Where("id = ?", id).First(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

// ListWebhooksByDomain 按域查询 Webhook 列表
func (s *AlertService) ListWebhooksByDomain(domainID string) ([]WebhookConfig, error) {
	var configs []WebhookConfig
	query := s.db
	if domainID != "" {
		query = query.Where("domain_id = ?", domainID)
	}
	if err := query.Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// UpdateWebhook 更新 Webhook 配置
func (s *AlertService) UpdateWebhook(id string, req CreateWebhookRequest) (*WebhookConfig, error) {
	config, err := s.GetWebhook(id)
	if err != nil {
		return nil, err
	}

	headersJSON := "{}"
	if req.Headers != nil {
		data, err := json.Marshal(req.Headers)
		if err != nil {
			return nil, err
		}
		headersJSON = string(data)
	}

	updates := map[string]interface{}{
		"name":       req.Name,
		"domain_id":  req.DomainID,
		"url":        req.URL,
		"method":     req.Method,
		"headers":    headersJSON,
		"filter":     req.Filter,
		"rate_limit": req.RateLimit,
		"updated_at": time.Now(),
	}

	if err := s.db.Model(config).Updates(updates).Error; err != nil {
		return nil, err
	}

	return s.GetWebhook(id)
}

// DeleteWebhook 删除 Webhook 配置
func (s *AlertService) DeleteWebhook(id string) error {
	return s.db.Where("id = ?", id).Delete(&WebhookConfig{}).Error
}

// TestWebhook 测试 Webhook 发送
func (s *AlertService) TestWebhook(id string) error {
	config, err := s.GetWebhook(id)
	if err != nil {
		return err
	}

	testEvent := &AlertEvent{
		AlertID:   "test_alert",
		DomainID:  config.DomainID,
		DeviceID:  "test_device",
		RuleID:    "test_rule",
		Level:     LevelInfo,
		Title:     "测试告警",
		Message:   "这是一条测试告警消息",
		Timestamp: time.Now().Unix(),
	}

	return s.sender.Send(config, testEvent)
}

// CountRecent 统计最近 24 小时告警数
func (s *AlertService) CountRecent() (int64, error) {
	var count int64
	since := time.Now().Add(-24 * time.Hour)
	if err := s.db.Model(&AlertLog{}).Where("created_at >= ?", since).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ListAlertLogs 查询告警日志
func (s *AlertService) ListAlertLogs(webhookID string) ([]AlertLog, error) {
	var logs []AlertLog
	query := s.db
	if webhookID != "" {
		query = query.Where("webhook_id = ?", webhookID)
	}
	if err := query.Order("created_at DESC").Limit(100).Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// ProcessAlertEvent 处理告警事件：匹配 webhook 配置并 fan-out 发送
func (s *AlertService) ProcessAlertEvent(event *AlertEvent) error {
	// 查询该域下所有 webhook
	configs, err := s.ListWebhooksByDomain(event.DomainID)
	if err != nil {
		return fmt.Errorf("failed to list webhooks: %w", err)
	}

	for _, config := range configs {
		// 检查级别过滤
		if len(config.Filter.Levels) > 0 {
			matched := false
			for _, level := range config.Filter.Levels {
				if level == event.Level {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// 发送 webhook
		status := "sent"
		response := ""
		if err := s.sender.Send(&config, event); err != nil {
			status = "failed"
			response = err.Error()
		}

		// 记录日志
		log := &AlertLog{
			AlertID:   event.AlertID,
			WebhookID: config.ID,
			Status:    status,
			Response:  response,
			CreatedAt: time.Now(),
		}
		s.db.Create(log)
	}

	return nil
}
