package device

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type AuthManager struct {
	rdb     *redis.Client
	service *Service
}

func NewAuthManager(rdb *redis.Client, service *Service) *AuthManager {
	return &AuthManager{
		rdb:     rdb,
		service: service,
	}
}

// Authenticate 设备认证
// 支持两种模式：
// 1. 设备级 Token 认证
// 2. 域级密钥认证（自动发现）
func (a *AuthManager) Authenticate(ctx context.Context, deviceID, credential string) (*Device, error) {
	// 先尝试设备级 Token 认证
	device, err := a.service.VerifyToken(deviceID, credential)
	if err == nil {
		return device, nil
	}

	// 再尝试域级密钥认证（自动发现模式）
	// credential 格式: domain_key:domain_id
	parts := strings.SplitN(credential, ":", 2)
	if len(parts) == 2 {
		domainKey := parts[0]
		domainID := parts[1]
		device, err = a.service.VerifyDomainKey(domainID, domainKey)
		if err == nil {
			return device, nil
		}
	}

	return nil, fmt.Errorf("authentication failed")
}

// CheckPermission 检查设备权限
func (a *AuthManager) CheckPermission(ctx context.Context, deviceID, topic, action string) bool {
	cacheKey := fmt.Sprintf("acl:%s", deviceID)

	// 尝试从缓存获取
	cached, err := a.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		return strings.Contains(cached, topic)
	}

	// 缓存未命中，检查设备是否存在
	device, err := a.service.GetByID(deviceID)
	if err != nil {
		return false
	}

	// 简单权限检查：设备只能操作自己的 topic
	// domains.{domain}.devices.>.>.>.{deviceID}.{direction}
	if strings.Contains(topic, device.ID) {
		// 缓存权限（5分钟）
		a.rdb.Set(ctx, cacheKey, topic, 5*time.Minute)
		return true
	}

	return false
}
