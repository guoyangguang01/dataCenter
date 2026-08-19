package model

import (
	"time"
)

// ThingModel 物模型定义
type ThingModel struct {
	ID         string         `json:"id" gorm:"primaryKey;size:64"`
	Name       string         `json:"name" gorm:"size:128"`
	DomainID   string         `json:"domain_id" gorm:"size:64;index"`
	Properties []PropertyDef  `json:"properties" gorm:"serializer:json"`
	Commands   []CommandDef   `json:"commands" gorm:"serializer:json"`
	Events     []EventDef     `json:"events" gorm:"serializer:json"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// PropertyDef 属性定义
type PropertyDef struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	DataType   string  `json:"data_type"` // float / int / string / bool / enum
	Unit       string  `json:"unit"`
	Range      [2]float64 `json:"range"`
	Required   bool    `json:"required"`
	AccessMode string  `json:"access_mode"` // r / rw
}

// CommandDef 命令定义
type CommandDef struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Params []ParamDef  `json:"params"`
}

// ParamDef 参数定义
type ParamDef struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	DataType string `json:"data_type"`
	Range    [2]float64 `json:"range,omitempty"`
}

// EventDef 事件定义
type EventDef struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Params []ParamDef  `json:"params"`
}

// DeviceModelBinding 设备-模型绑定
type DeviceModelBinding struct {
	DeviceID string `json:"device_id" gorm:"primaryKey;size:64"`
	ModelID  string `json:"model_id" gorm:"size:64;index"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateModelRequest 创建物模型请求
type CreateModelRequest struct {
	ID         string        `json:"id" binding:"required"`
	Name       string        `json:"name" binding:"required"`
	DomainID   string        `json:"domain_id" binding:"required"`
	Properties []PropertyDef `json:"properties"`
	Commands   []CommandDef  `json:"commands"`
	Events     []EventDef    `json:"events"`
}

// BindModelRequest 绑定模型请求
type BindModelRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	ModelID  string `json:"model_id" binding:"required"`
}
