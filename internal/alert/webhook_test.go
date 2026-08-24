package alert

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestEvent(deviceID, ruleID string) *AlertEvent {
	return &AlertEvent{
		AlertID:  "alert-001",
		DeviceID: deviceID,
		RuleID:   ruleID,
		Level:    LevelWarning,
		Title:    "Test Alert",
		Message:  "Temperature too high",
	}
}

func TestWebhookSender_Send(t *testing.T) {
	// 启动测试 HTTP 服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewWebhookSender()
	config := &WebhookConfig{
		ID:     "wh-001",
		Name:   "test-webhook",
		URL:    server.URL,
		Method: "POST",
	}

	err := sender.Send(config, newTestEvent("dev-001", "rule-001"))
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
}

func TestWebhookSender_Dedup(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewWebhookSender()
	config := &WebhookConfig{
		ID:     "wh-001",
		Name:   "test-webhook",
		URL:    server.URL,
		Method: "POST",
	}
	event := newTestEvent("dev-001", "rule-001")

	// 第一次发送成功
	if err := sender.Send(config, event); err != nil {
		t.Fatalf("first Send failed: %v", err)
	}

	// 第二次相同事件应该被去重
	err := sender.Send(config, event)
	if err == nil {
		t.Errorf("expected dedup error for duplicate event")
	}

	if callCount != 1 {
		t.Errorf("expected 1 HTTP call, got %d", callCount)
	}
}

func TestWebhookSender_DedupExpiry(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewWebhookSender()
	config := &WebhookConfig{
		ID:     "wh-001",
		Name:   "test-webhook",
		URL:    server.URL,
		Method: "POST",
	}
	event := newTestEvent("dev-001", "rule-001")

	// 手动设置过期的去重记录
	key := "wh-001:dev-001:rule-001"
	sender.dedupCache[key] = time.Now().Add(-6 * time.Minute) // 超过 5 分钟窗口

	// 应该能再次发送
	if err := sender.Send(config, event); err != nil {
		t.Fatalf("Send should succeed after dedup expiry: %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected 1 HTTP call, got %d", callCount)
	}
}

func TestWebhookSender_RateLimit(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewWebhookSender()
	config := &WebhookConfig{
		ID:     "wh-001",
		Name:   "test-webhook",
		URL:    server.URL,
		Method: "POST",
	}

	// 发送 10 条不同事件（达到限流阈值）
	for i := 0; i < 10; i++ {
		event := newTestEvent("dev-"+string(rune('A'+i)), "rule-001")
		err := sender.Send(config, event)
		if err != nil {
			t.Fatalf("Send %d failed: %v", i, err)
		}
	}

	// 第 11 条应该被限流
	event := newTestEvent("dev-extra", "rule-001")
	err := sender.Send(config, event)
	if err == nil {
		t.Errorf("expected rate limit error")
	}

	if callCount != 10 {
		t.Errorf("expected 10 HTTP calls, got %d", callCount)
	}
}

func TestWebhookSender_RateLimitWindowReset(t *testing.T) {
	sender := NewWebhookSender()

	// 手动设置过期的限流记录
	sender.rateLimiter["wh-001"] = []time.Time{
		time.Now().Add(-2 * time.Minute),
		time.Now().Add(-90 * time.Second),
	}

	// 窗口已过期，不应该被限流
	limited := sender.isRateLimited("wh-001")
	if limited {
		t.Errorf("expected not rate limited after window reset")
	}
}

func TestWebhookSender_WebhookError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	sender := NewWebhookSender()
	config := &WebhookConfig{
		ID:     "wh-001",
		Name:   "test-webhook",
		URL:    server.URL,
		Method: "POST",
	}

	err := sender.Send(config, newTestEvent("dev-001", "rule-001"))
	if err == nil {
		t.Errorf("expected error for 500 response")
	}
}

func TestWebhookSender_ServerDown(t *testing.T) {
	sender := NewWebhookSender()
	config := &WebhookConfig{
		ID:     "wh-001",
		Name:   "test-webhook",
		URL:    "http://localhost:1", // 不可达
		Method: "POST",
	}

	err := sender.Send(config, newTestEvent("dev-001", "rule-001"))
	if err == nil {
		t.Errorf("expected error for unreachable server")
	}
}
