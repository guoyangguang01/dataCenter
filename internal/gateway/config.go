package gateway

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// GatewayType 网关类型
type GatewayType string

const (
	TypeMQTT   GatewayType = "mqtt"
	TypeTCP    GatewayType = "tcp"
	TypeModbus GatewayType = "modbus"
	TypeOPCUA  GatewayType = "opcua"
)

// GatewayStatus 网关运行状态
type GatewayStatus string

const (
	StatusStopped GatewayStatus = "stopped"
	StatusRunning GatewayStatus = "running"
	StatusError   GatewayStatus = "error"
)

// GatewayConfig 网关配置（GORM 模型）
type GatewayConfig struct {
	ID        string        `json:"id" gorm:"primaryKey;size:64"`
	Name      string        `json:"name" gorm:"size:128"`
	Type      GatewayType   `json:"type" gorm:"size:16;index"`
	Config    string        `json:"config" gorm:"type:text"` // JSON 编码的协议专属配置
	Enabled   bool          `json:"enabled" gorm:"default:true"`
	Status    GatewayStatus `json:"status" gorm:"size:16;default:stopped"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// MQTTConfig MQTT 网关配置
type MQTTConfig struct {
	Port          int `json:"port"`
	MaxConnection int `json:"max_connection"`
	KeepAlive     int `json:"keep_alive"`
}

// TCPConfig TCP 网关配置
type TCPConfig struct {
	Port          int `json:"port"`
	MaxConnection int `json:"max_connection"`
	Heartbeat     int `json:"heartbeat"`
}

// ModbusConfig Modbus 网关配置
type ModbusConfig struct {
	Port         int   `json:"port"`
	PollInterval int   `json:"poll_interval"`
	SlaveIDs     []int `json:"slave_ids"`
}

// OPCUAConfig OPC UA 网关配置
type OPCUAConfig struct {
	Endpoint     string   `json:"endpoint"`
	PollInterval int      `json:"poll_interval"`
	NodeIDs      []string `json:"node_ids"`
	DeviceID     string   `json:"device_id"`
	DomainID     string   `json:"domain_id"`
}

// CreateGatewayRequest 创建网关请求
type CreateGatewayRequest struct {
	Name   string      `json:"name" binding:"required"`
	Type   GatewayType `json:"type" binding:"required"`
	Config interface{} `json:"config" binding:"required"`
}

// GatewayService 网关配置管理服务
type GatewayService struct {
	db *gorm.DB
}

// NewGatewayService 创建网关配置服务
func NewGatewayService(db *gorm.DB) *GatewayService {
	return &GatewayService{db: db}
}

// InitDB 初始化数据库表
func (s *GatewayService) InitDB() error {
	return s.db.AutoMigrate(&GatewayConfig{})
}

// Create 创建网关配置
func (s *GatewayService) Create(req CreateGatewayRequest) (*GatewayConfig, error) {
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return nil, err
	}

	gc := &GatewayConfig{
		ID:        fmt.Sprintf("gw_%d", time.Now().UnixNano()),
		Name:      req.Name,
		Type:      req.Type,
		Config:    string(configJSON),
		Enabled:   true,
		Status:    StatusStopped,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.db.Create(gc).Error; err != nil {
		return nil, err
	}
	return gc, nil
}

// GetByID 获取网关配置
func (s *GatewayService) GetByID(id string) (*GatewayConfig, error) {
	var gc GatewayConfig
	if err := s.db.Where("id = ?", id).First(&gc).Error; err != nil {
		return nil, err
	}
	return &gc, nil
}

// List 查询网关配置列表
func (s *GatewayService) List() ([]GatewayConfig, error) {
	var configs []GatewayConfig
	if err := s.db.Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// Update 更新网关配置
func (s *GatewayService) Update(id string, req CreateGatewayRequest) (*GatewayConfig, error) {
	gc, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"name":       req.Name,
		"type":       req.Type,
		"config":     string(configJSON),
		"updated_at": time.Now(),
	}

	if err := s.db.Model(gc).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetByID(id)
}

// Delete 删除网关配置
func (s *GatewayService) Delete(id string) error {
	return s.db.Where("id = ?", id).Delete(&GatewayConfig{}).Error
}

// UpdateStatus 更新网关运行状态
func (s *GatewayService) UpdateStatus(id string, status GatewayStatus) error {
	return s.db.Model(&GatewayConfig{}).Where("id = ?", id).Update("status", status).Error
}
