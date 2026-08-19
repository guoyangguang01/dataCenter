package device

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// InitDB 初始化数据库表
func (s *Service) InitDB() error {
	return s.db.AutoMigrate(&Device{})
}

// Create 创建设备
func (s *Service) Create(req CreateDeviceRequest) (*Device, error) {
	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	device := &Device{
		ID:         req.ID,
		Name:       req.Name,
		DeviceType: req.DeviceType,
		Protocol:   req.Protocol,
		DomainID:   req.DomainID,
		Region:     req.Region,
		ModelID:    req.ModelID,
		Firmware:   req.Firmware,
		Token:      token,
		Status:     StatusOffline,
		Online:     false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.db.Create(device).Error; err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	return device, nil
}

// GetByID 根据 ID 获取设备
func (s *Service) GetByID(id string) (*Device, error) {
	var device Device
	if err := s.db.Where("id = ?", id).First(&device).Error; err != nil {
		return nil, err
	}
	return &device, nil
}

// Update 更新设备
func (s *Service) Update(id string, req UpdateDeviceRequest) (*Device, error) {
	device, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.DeviceType != nil {
		updates["device_type"] = *req.DeviceType
	}
	if req.Region != nil {
		updates["region"] = *req.Region
	}
	if req.ModelID != nil {
		updates["model_id"] = *req.ModelID
	}
	if req.Firmware != nil {
		updates["firmware"] = *req.Firmware
	}

	if err := s.db.Model(device).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update device: %w", err)
	}

	return s.GetByID(id)
}

// Delete 删除设备
func (s *Service) Delete(id string) error {
	return s.db.Where("id = ?", id).Delete(&Device{}).Error
}

// List 查询设备列表
func (s *Service) List(query DeviceQuery) ([]Device, int64, error) {
	var devices []Device
	var total int64

	db := s.db.Model(&Device{})

	if query.DomainID != "" {
		db = db.Where("domain_id = ?", query.DomainID)
	}
	if query.DeviceType != "" {
		db = db.Where("device_type = ?", query.DeviceType)
	}
	if query.Protocol != "" {
		db = db.Where("protocol = ?", query.Protocol)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PageSize
	if err := db.Offset(offset).Limit(query.PageSize).Find(&devices).Error; err != nil {
		return nil, 0, err
	}

	return devices, total, nil
}

// VerifyToken 验证设备 Token
func (s *Service) VerifyToken(deviceID, token string) (*Device, error) {
	var device Device
	if err := s.db.Where("id = ? AND token = ?", deviceID, token).First(&device).Error; err != nil {
		return nil, fmt.Errorf("invalid token")
	}
	return &device, nil
}

// VerifyDomainKey 验证域密钥（自动发现模式）
func (s *Service) VerifyDomainKey(domainID, key string) (*Device, error) {
	var device Device
	if err := s.db.Where("domain_id = ? AND domain_key = ?", domainID, key).First(&device).Error; err != nil {
		return nil, fmt.Errorf("invalid domain key")
	}
	return &device, nil
}

func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
