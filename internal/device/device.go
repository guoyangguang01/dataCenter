package device

import (
	"time"
)

// DeviceStatus 设备状态枚举
type DeviceStatus int

const (
	StatusOffline DeviceStatus = 0
	StatusOnline  DeviceStatus = 1
)

// Device 设备档案
type Device struct {
	ID          string            `json:"id" gorm:"primaryKey;size:64"`
	Name        string            `json:"name" gorm:"size:128"`
	DeviceType  string            `json:"device_type" gorm:"size:32;index"`
	Protocol    string            `json:"protocol" gorm:"size:16;index"`
	DomainID    string            `json:"domain_id" gorm:"size:64;index"`
	Region      string            `json:"region" gorm:"size:32"`
	ModelID     string            `json:"model_id" gorm:"size:64;index"`
	Firmware    string            `json:"firmware" gorm:"size:32"`
	Token       string            `json:"token" gorm:"size:128"`
	DomainKey   string            `json:"domain_key" gorm:"size:128"`
	Tags        string            `json:"tags" gorm:"type:text"` // JSON encoded
	Status      DeviceStatus      `json:"status" gorm:"default:0"`
	Online      bool              `json:"online" gorm:"default:false"`
	LastSeen    time.Time         `json:"last_seen"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	DeletedAt   *time.Time        `json:"deleted_at,omitempty" gorm:"index"`
}

// DeviceStatusInfo 设备实时状态（存 Redis）
type DeviceStatusInfo struct {
	DeviceID  string    `json:"device_id"`
	Online    bool      `json:"online"`
	LastSeen  time.Time `json:"last_seen"`
	IP        string    `json:"ip"`
	RSSI      int       `json:"rssi"`
	Battery   int       `json:"battery"`
	Uptime    int64     `json:"uptime"`
}

// CreateDeviceRequest 创建设备请求
type CreateDeviceRequest struct {
	ID         string            `json:"id" binding:"required"`
	Name       string            `json:"name" binding:"required"`
	DeviceType string            `json:"device_type" binding:"required"`
	Protocol   string            `json:"protocol" binding:"required"`
	DomainID   string            `json:"domain_id" binding:"required"`
	Region     string            `json:"region"`
	ModelID    string            `json:"model_id"`
	Firmware   string            `json:"firmware"`
	Tags       map[string]string `json:"tags"`
}

// UpdateDeviceRequest 更新设备请求
type UpdateDeviceRequest struct {
	Name       *string           `json:"name"`
	DeviceType *string           `json:"device_type"`
	Region     *string           `json:"region"`
	ModelID    *string           `json:"model_id"`
	Firmware   *string           `json:"firmware"`
	Tags       map[string]string `json:"tags"`
}

// DeviceQuery 设备查询参数
type DeviceQuery struct {
	DomainID   string `form:"domain_id"`
	DeviceType string `form:"device_type"`
	Protocol   string `form:"protocol"`
	Online     *bool  `form:"online"`
	Page       int    `form:"page" binding:"required,min=1"`
	PageSize   int    `form:"page_size" binding:"required,min=1,max=100"`
}
