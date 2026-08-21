package model

import (
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
	return s.db.AutoMigrate(&ThingModel{}, &DeviceModelBinding{})
}

// Create 创建物模型
func (s *Service) Create(req CreateModelRequest) (*ThingModel, error) {
	model := &ThingModel{
		ID:         req.ID,
		Name:       req.Name,
		DomainID:   req.DomainID,
		Properties: req.Properties,
		Commands:   req.Commands,
		Events:     req.Events,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.db.Create(model).Error; err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}
	return model, nil
}

// GetByID 根据 ID 获取模型
func (s *Service) GetByID(id string) (*ThingModel, error) {
	var model ThingModel
	if err := s.db.Where("id = ?", id).First(&model).Error; err != nil {
		return nil, err
	}
	return &model, nil
}

// ListAll 查询所有模型
func (s *Service) ListAll() ([]ThingModel, error) {
	var models []ThingModel
	if err := s.db.Find(&models).Error; err != nil {
		return nil, err
	}
	return models, nil
}

// ListByDomain 按域查询模型列表
func (s *Service) ListByDomain(domainID string) ([]ThingModel, error) {
	var models []ThingModel
	if err := s.db.Where("domain_id = ?", domainID).Find(&models).Error; err != nil {
		return nil, err
	}
	return models, nil
}

// Delete 删除模型
func (s *Service) Delete(id string) error {
	return s.db.Where("id = ?", id).Delete(&ThingModel{}).Error
}

// BindDevice 绑定设备到模型
func (s *Service) BindDevice(deviceID, modelID string) error {
	binding := &DeviceModelBinding{
		DeviceID:  deviceID,
		ModelID:   modelID,
		CreatedAt: time.Now(),
	}
	return s.db.Create(binding).Error
}

// GetDeviceModel 获取设备绑定的模型
func (s *Service) GetDeviceModel(deviceID string) (*ThingModel, error) {
	var binding DeviceModelBinding
	if err := s.db.Where("device_id = ?", deviceID).First(&binding).Error; err != nil {
		return nil, err
	}
	return s.GetByID(binding.ModelID)
}

// UnbindDevice 解绑设备
func (s *Service) UnbindDevice(deviceID string) error {
	return s.db.Where("device_id = ?", deviceID).Delete(&DeviceModelBinding{}).Error
}

// ValidateData 根据物模型校验数据
func (s *Service) ValidateData(modelID, topic string, value float64) error {
	model, err := s.GetByID(modelID)
	if err != nil {
		return err
	}

	for _, prop := range model.Properties {
		if prop.ID == topic {
			if value < prop.Range[0] || value > prop.Range[1] {
				return fmt.Errorf("value %f out of range [%f, %f]", value, prop.Range[0], prop.Range[1])
			}
			return nil
		}
	}
	return fmt.Errorf("property %s not found in model", topic)
}
