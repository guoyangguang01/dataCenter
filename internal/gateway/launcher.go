package gateway

import (
	"encoding/json"
	"fmt"
	"sync"
)

// GatewayFactory 网关工厂函数
type GatewayFactory func(config string, publisher Publisher) (GatewayAdapter, error)

// Launcher 网关启动器
type Launcher struct {
	service   *GatewayService
	publisher Publisher
	factories map[GatewayType]GatewayFactory
	running   map[string]GatewayAdapter
	mu        sync.RWMutex
}

// NewLauncher 创建网关启动器（重置所有网关状态为 stopped）
func NewLauncher(service *GatewayService, publisher Publisher) *Launcher {
	// 后端重启后，内存中的 running 状态丢失，重置数据库状态
	service.ResetAllStatus()
	return &Launcher{
		service:   service,
		publisher: publisher,
		factories: make(map[GatewayType]GatewayFactory),
		running:   make(map[string]GatewayAdapter),
	}
}

// Register 注册网关工厂
func (l *Launcher) Register(gwType GatewayType, factory GatewayFactory) {
	l.factories[gwType] = factory
}

// StartGateway 启动网关
func (l *Launcher) StartGateway(id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 检查是否已在运行
	if _, ok := l.running[id]; ok {
		return fmt.Errorf("gateway %s is already running", id)
	}

	gc, err := l.service.GetByID(id)
	if err != nil {
		return err
	}

	factory, ok := l.factories[gc.Type]
	if !ok {
		return fmt.Errorf("unsupported gateway type: %s", gc.Type)
	}

	adapter, err := factory(gc.Config, l.publisher)
	if err != nil {
		l.service.UpdateStatus(id, StatusError)
		return fmt.Errorf("failed to create gateway: %w", err)
	}

	if err := adapter.Start(); err != nil {
		l.service.UpdateStatus(id, StatusError)
		return fmt.Errorf("failed to start gateway: %w", err)
	}

	l.running[id] = adapter
	l.service.UpdateStatus(id, StatusRunning)
	fmt.Printf("[Launcher] gateway %s (%s) started\n", gc.Name, gc.Type)
	return nil
}

// StopGateway 停止网关
func (l *Launcher) StopGateway(id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	adapter, ok := l.running[id]
	if !ok {
		// 网关不在内存中（可能重启后遗留），直接更新数据库状态
		fmt.Printf("[Launcher] gateway %s not in memory, updating db status\n", id)
		l.service.UpdateStatus(id, StatusStopped)
		return nil
	}

	adapter.Stop()
	delete(l.running, id)
	l.service.UpdateStatus(id, StatusStopped)
	fmt.Printf("[Launcher] gateway %s stopped\n", id)
	return nil
}

// StartAll 启动所有已启用的网关
func (l *Launcher) StartAll() {
	configs, err := l.service.List()
	if err != nil {
		fmt.Printf("[Launcher] failed to load configs: %v\n", err)
		return
	}

	for _, gc := range configs {
		if gc.Enabled {
			if err := l.StartGateway(gc.ID); err != nil {
				fmt.Printf("[Launcher] failed to start %s: %v\n", gc.Name, err)
			}
		}
	}
}

// StopAll 停止所有网关
func (l *Launcher) StopAll() {
	l.mu.Lock()
	ids := make([]string, 0, len(l.running))
	for id := range l.running {
		ids = append(ids, id)
	}
	l.mu.Unlock()

	for _, id := range ids {
		l.StopGateway(id)
	}
}

// SendToDevice 向指定设备发送下行命令
// 遍历所有运行中的网关查找设备所在位置
func (l *Launcher) SendToDevice(deviceID string, payload []byte) error {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, gw := range l.running {
		devices := gw.GetConnectedDevices()
		for _, d := range devices {
			if d == deviceID {
				return gw.SendCommand(deviceID, payload)
			}
		}
	}

	return fmt.Errorf("device %s not connected to any gateway", deviceID)
}

// GetConnectedDevices 返回所有网关的已连接设备列表
func (l *Launcher) GetConnectedDevices() map[string][]string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make(map[string][]string)
	for id, gw := range l.running {
		devices := gw.GetConnectedDevices()
		if len(devices) > 0 {
			result[id] = devices
		}
	}
	return result
}

// RunningCount 返回正在运行的网关数量
func (l *Launcher) RunningCount() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.running)
}

// IsRunning 检查网关是否在运行
func (l *Launcher) IsRunning(id string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	_, ok := l.running[id]
	return ok
}

// ParseMQTTConfig 解析 MQTT 配置
func ParseMQTTConfig(configStr string) (*MQTTConfig, error) {
	var cfg MQTTConfig
	if err := json.Unmarshal([]byte(configStr), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ParseTCPConfig 解析 TCP 配置
func ParseTCPConfig(configStr string) (*TCPConfig, error) {
	var cfg TCPConfig
	if err := json.Unmarshal([]byte(configStr), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ParseModbusConfig 解析 Modbus 配置
func ParseModbusConfig(configStr string) (*ModbusConfig, error) {
	var cfg ModbusConfig
	if err := json.Unmarshal([]byte(configStr), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ParseOPCUAConfig 解析 OPC UA 配置
func ParseOPCUAConfig(configStr string) (*OPCUAConfig, error) {
	var cfg OPCUAConfig
	if err := json.Unmarshal([]byte(configStr), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
