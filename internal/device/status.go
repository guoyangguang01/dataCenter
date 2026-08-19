package device

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type StatusManager struct {
	rdb        *redis.Client
	offlineThreshold time.Duration
}

func NewStatusManager(rdb *redis.Client) *StatusManager {
	return &StatusManager{
		rdb:              rdb,
		offlineThreshold: 3 * time.Minute,
	}
}

func (m *StatusManager) key(deviceID string) string {
	return fmt.Sprintf("device:status:%s", deviceID)
}

// UpdateOnline 更新设备在线状态
func (m *StatusManager) UpdateOnline(ctx context.Context, deviceID string, online bool) error {
	status := DeviceStatusInfo{
		DeviceID: deviceID,
		Online:   online,
		LastSeen: time.Now(),
	}
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}
	return m.rdb.Set(ctx, m.key(deviceID), data, 24*time.Hour).Err()
}

// UpdateLastSeen 更新最后活跃时间
func (m *StatusManager) UpdateLastSeen(ctx context.Context, deviceID string) error {
	data, err := m.rdb.Get(ctx, m.key(deviceID)).Bytes()
	if err != nil {
		// 不存在则创建
		status := DeviceStatusInfo{
			DeviceID: deviceID,
			Online:   true,
			LastSeen: time.Now(),
		}
		data, _ = json.Marshal(status)
		return m.rdb.Set(ctx, m.key(deviceID), data, 24*time.Hour).Err()
	}

	var status DeviceStatusInfo
	if err := json.Unmarshal(data, &status); err != nil {
		return err
	}
	status.LastSeen = time.Now()
	status.Online = true

	data, _ = json.Marshal(status)
	return m.rdb.Set(ctx, m.key(deviceID), data, 24*time.Hour).Err()
}

// GetStatus 获取设备状态
func (m *StatusManager) GetStatus(ctx context.Context, deviceID string) (*DeviceStatusInfo, error) {
	data, err := m.rdb.Get(ctx, m.key(deviceID)).Bytes()
	if err != nil {
		return nil, err
	}
	var status DeviceStatusInfo
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// IsOnline 检查设备是否在线
func (m *StatusManager) IsOnline(ctx context.Context, deviceID string) bool {
	status, err := m.GetStatus(ctx, deviceID)
	if err != nil {
		return false
	}
	return status.Online && time.Since(status.LastSeen) < m.offlineThreshold
}

// ScanOffline 扫描离线设备（定时任务调用）
func (m *StatusManager) ScanOffline(ctx context.Context) ([]string, error) {
	pattern := "device:status:*"
	var offlineDevices []string

	iter := m.rdb.Scan(ctx, 0, pattern, 1000).Iterator()
	for iter.Next(ctx) {
		data, err := m.rdb.Get(ctx, iter.Val()).Bytes()
		if err != nil {
			continue
		}
		var status DeviceStatusInfo
		if err := json.Unmarshal(data, &status); err != nil {
			continue
		}
		if status.Online && time.Since(status.LastSeen) > m.offlineThreshold {
			offlineDevices = append(offlineDevices, status.DeviceID)
			// 标记为离线
			status.Online = false
			newData, _ := json.Marshal(status)
			m.rdb.Set(ctx, iter.Val(), newData, 24*time.Hour)
		}
	}
	return offlineDevices, iter.Err()
}
