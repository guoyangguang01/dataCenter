package simulator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/datacenter/internal/simulator/devices"
	"github.com/rs/zerolog"
)

// ProtocolAdapter defines the interface for protocol-specific adapters
type ProtocolAdapter interface {
	Start(ctx context.Context) error
	Stop() error
	SendData(deviceID string, data map[string]interface{}) error
}

// SimulatorEngine is the core simulator engine
type SimulatorEngine struct {
	protocol  string
	scenario  *ScenarioDefinition
	templates map[string]*devices.DeviceTemplateLoaded
	adapter   ProtocolAdapter
	devices   []*devices.DeviceInstance
	logger    zerolog.Logger
	quit      chan struct{}
	wg        sync.WaitGroup
}

// NewSimulatorEngine creates a new simulator engine
func NewSimulatorEngine(
	protocol string,
	scenario *ScenarioDefinition,
	templates map[string]*devices.DeviceTemplateLoaded,
	adapter ProtocolAdapter,
	logger zerolog.Logger,
) *SimulatorEngine {
	return &SimulatorEngine{
		protocol:  protocol,
		scenario:  scenario,
		templates: templates,
		adapter:   adapter,
		logger:    logger,
		quit:      make(chan struct{}),
	}
}

// Start starts the simulator engine
func (e *SimulatorEngine) Start(ctx context.Context) error {
	e.logger.Info().
		Str("protocol", e.protocol).
		Str("scenario", e.scenario.Name).
		Msg("Starting simulator engine")

	// Initialize devices
	if err := e.initDevices(); err != nil {
		return fmt.Errorf("failed to initialize devices: %w", err)
	}

	// Start protocol adapter
	if err := e.adapter.Start(ctx); err != nil {
		return fmt.Errorf("failed to start adapter: %w", err)
	}

	// Parse runtime config
	startDelay := GetDuration(e.scenario.Runtime.StartDelay, 0)
	stagger := GetDuration(e.scenario.Runtime.Stagger, 100*time.Millisecond)
	duration := GetDuration(e.scenario.Runtime.Duration, 0)

	// Start device simulators
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.runDeviceSimulators(ctx, startDelay, stagger, duration)
	}()

	e.logger.Info().
		Int("device_count", len(e.devices)).
		Msg("Simulator engine started")

	return nil
}

// Stop stops the simulator engine
func (e *SimulatorEngine) Stop() {
	e.logger.Info().Msg("Stopping simulator engine")
	close(e.quit)
	e.wg.Wait()

	if err := e.adapter.Stop(); err != nil {
		e.logger.Error().Err(err).Msg("Failed to stop adapter")
	}

	e.logger.Info().Msg("Simulator engine stopped")
}

// initDevices initializes device instances from configuration
func (e *SimulatorEngine) initDevices() error {
	for _, devCfg := range e.scenario.Devices {
		template, ok := e.templates[devCfg.Template]
		if !ok {
			return fmt.Errorf("template %s not found", devCfg.Template)
		}

		for i := 0; i < devCfg.Count; i++ {
			deviceID := devices.FormatID(devCfg.IDPrefix, i)
			instance := devices.NewDeviceInstance(
				deviceID,
				devCfg.DomainID,
				devCfg.Region,
				e.protocol,
				template,
			)
			e.devices = append(e.devices, instance)

			e.logger.Info().
				Str("device_id", deviceID).
				Str("template", devCfg.Template).
				Msg("Device instance created")
		}
	}

	return nil
}

// runDeviceSimulators runs all device simulators
func (e *SimulatorEngine) runDeviceSimulators(
	ctx context.Context,
	startDelay, stagger, duration time.Duration,
) {
	// Apply start delay
	if startDelay > 0 {
		e.logger.Info().Dur("delay", startDelay).Msg("Waiting before start")
		select {
		case <-time.After(startDelay):
		case <-e.quit:
			return
		}
	}

	// Create a deadline context if duration is specified
	var cancel context.CancelFunc
	if duration > 0 {
		ctx, cancel = context.WithTimeout(ctx, duration)
		defer cancel()
	}

	// Start each device with stagger
	for i, dev := range e.devices {
		if i > 0 && stagger > 0 {
			select {
			case <-time.After(stagger):
			case <-e.quit:
				return
			}
		}

		e.wg.Add(1)
		go func(d *devices.DeviceInstance) {
			defer e.wg.Done()
			e.runDevice(ctx, d)
		}(dev)
	}

	// Wait for context cancellation
	<-ctx.Done()
}

// runDevice runs a single device simulator
func (e *SimulatorEngine) runDevice(ctx context.Context, dev *devices.DeviceInstance) {
	e.logger.Info().
		Str("device_id", dev.DeviceID).
		Dur("interval", dev.ReportInterval).
		Msg("[Sim] 设备模拟器启动")

	ticker := time.NewTicker(dev.ReportInterval)
	defer ticker.Stop()

	// Send initial data immediately
	e.logger.Info().
		Str("device_id", dev.DeviceID).
		Msg("[Sim] 发送初始数据...")
	e.sendDeviceData(dev)

	sendCount := 1
	for {
		select {
		case <-ticker.C:
			sendCount++
			e.logger.Info().
				Str("device_id", dev.DeviceID).
				Int("send_count", sendCount).
				Msg("[Sim] 定时器触发，准备发送数据")
			e.sendDeviceData(dev)
		case <-ctx.Done():
			e.logger.Info().
				Str("device_id", dev.DeviceID).
				Int("total_sent", sendCount).
				Msg("[Sim] 设备模拟器停止（context 取消）")
			return
		case <-e.quit:
			e.logger.Info().
				Str("device_id", dev.DeviceID).
				Int("total_sent", sendCount).
				Msg("[Sim] 设备模拟器停止（quit 信号）")
			return
		}
	}
}

// sendDeviceData sends data from a device
func (e *SimulatorEngine) sendDeviceData(dev *devices.DeviceInstance) {
	data := dev.GenerateDataWithTimestamp(time.Now())

	dataJSON, _ := json.Marshal(data)
	e.logger.Info().
		Str("device_id", dev.DeviceID).
		Str("domain_id", dev.DomainID).
		Int("data_points", len(data)).
		Str("data", string(dataJSON)).
		Msg("[Sim] 数据生成完成，准备发送")

	if err := e.adapter.SendData(dev.DeviceID, data); err != nil {
		e.logger.Error().
			Err(err).
			Str("device_id", dev.DeviceID).
			Msg("[Sim] ❌ 数据发送失败，模拟器将停止")
		// Exit on send failure (no reconnection)
		e.quit <- struct{}{}
		return
	}

	e.logger.Info().
		Str("device_id", dev.DeviceID).
		Msg("[Sim] ✅ 数据发送成功")
}

// GetDeviceCount returns the number of active devices
func (e *SimulatorEngine) GetDeviceCount() int {
	return len(e.devices)
}

// GetDevices returns all device instances
func (e *SimulatorEngine) GetDevices() []*devices.DeviceInstance {
	return e.devices
}
