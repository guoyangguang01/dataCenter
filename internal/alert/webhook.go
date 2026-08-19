package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type WebhookSender struct {
	httpClient  *http.Client
	dedupCache   map[string]time.Time
	rateLimiter map[string][]time.Time
	mu          sync.RWMutex
}

func NewWebhookSender() *WebhookSender {
	return &WebhookSender{
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		dedupCache:  make(map[string]time.Time),
		rateLimiter: make(map[string][]time.Time),
	}
}

// Send 发送 Webhook
func (s *WebhookSender) Send(config *WebhookConfig, event *AlertEvent) error {
	// 检查去重
	if s.isDuplicate(config.ID, event) {
		return fmt.Errorf("duplicate alert")
	}

	// 检查限流
	if s.isRateLimited(config.ID) {
		return fmt.Errorf("rate limited")
	}

	// 构建请求体
	body, err := s.buildPayload(config, event)
	if err != nil {
		return err
	}

	// 发送请求
	req, err := http.NewRequest(config.Method, config.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned %d: %s", resp.StatusCode, string(respBody))
	}

	fmt.Println("[Alert] webhook sent:", config.Name)
	return nil
}

func (s *WebhookSender) isDuplicate(webhookID string, event *AlertEvent) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%s:%s:%s", webhookID, event.DeviceID, event.RuleID)
	if lastSeen, ok := s.dedupCache[key]; ok {
		if time.Since(lastSeen) < 5*time.Minute {
			return true
		}
	}
	s.dedupCache[key] = time.Now()
	return false
}

func (s *WebhookSender) isRateLimited(webhookID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	window := now.Add(-1 * time.Minute)

	// 清理过期记录
	requests := s.rateLimiter[webhookID]
	valid := make([]time.Time, 0)
	for _, t := range requests {
		if t.After(window) {
			valid = append(valid, t)
		}
	}
	s.rateLimiter[webhookID] = valid

	if len(valid) >= 10 { // 默认每分钟 10 条
		return true
	}

	s.rateLimiter[webhookID] = append(valid, now)
	return false
}

func (s *WebhookSender) buildPayload(config *WebhookConfig, event *AlertEvent) ([]byte, error) {
	// 默认钉钉格式
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": event.Title,
			"text":  fmt.Sprintf("**%s** %s 设备:%s 时间:%s", event.Level, event.Title, event.DeviceID, time.Now().Format("2006-01-02 15:04:05")),
		},
	}
	return json.Marshal(payload)
}
