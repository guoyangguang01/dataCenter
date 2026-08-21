package simulator

import (
	"fmt"
	"os"
	"time"

	"github.com/datacenter/internal/simulator/devices"
	"gopkg.in/yaml.v3"
)

// SimulatorConfig is the top-level simulator configuration
type SimulatorConfig struct {
	Simulator GlobalConfig `yaml:"simulator"`
}

// GlobalConfig holds global simulator settings
type GlobalConfig struct {
	LogLevel  string    `yaml:"log_level"`
	LogFormat string    `yaml:"log_format"`
	MQTT      MQTTGlobalConfig `yaml:"mqtt"`
	TCP       TCPGlobalConfig  `yaml:"tcp"`
	Modbus    ModbusGlobalConfig `yaml:"modbus"`
	OPCUA     OPCUAGlobalConfig `yaml:"opcua"`
}

// MQTTGlobalConfig holds global MQTT settings
type MQTTGlobalConfig struct {
	Broker         string `yaml:"broker"`
	Username       string `yaml:"username"`
	Password       string `yaml:"password"`
	ClientIDPrefix string `yaml:"client_id_prefix"`
}

// TCPGlobalConfig holds global TCP settings
type TCPGlobalConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// ModbusGlobalConfig holds global Modbus settings
type ModbusGlobalConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// OPCUAGlobalConfig holds global OPC UA settings
type OPCUAGlobalConfig struct {
	Endpoint string `yaml:"endpoint"`
}

// ScenarioConfig defines a simulation scenario
type ScenarioConfig struct {
	Scenario ScenarioDefinition `yaml:"scenario"`
}

// ScenarioDefinition holds scenario definition
type ScenarioDefinition struct {
	Name     string            `yaml:"name"`
	Protocol string            `yaml:"protocol"`
	Enabled  bool              `yaml:"enabled"`
	Devices  []DeviceConfig    `yaml:"devices"`
	MQTT     *MQTTScenarioConfig `yaml:"mqtt,omitempty"`
	TCP      *TCPScenarioConfig  `yaml:"tcp,omitempty"`
	Modbus   *ModbusScenarioConfig `yaml:"modbus,omitempty"`
	OPCUA    *OPCUAScenarioConfig `yaml:"opcua,omitempty"`
	Runtime  RuntimeConfig     `yaml:"runtime"`
	Logging  LoggingConfig     `yaml:"logging"`
}

// DeviceConfig defines devices to simulate
type DeviceConfig struct {
	Template string `yaml:"template"`
	Count    int    `yaml:"count"`
	IDPrefix string `yaml:"id_prefix"`
	DomainID string `yaml:"domain_id"`
	Region   string `yaml:"region"`
}

// MQTTScenarioConfig holds MQTT-specific scenario config
type MQTTScenarioConfig struct {
	Broker         string `yaml:"broker"`
	Username       string `yaml:"username"`
	Password       string `yaml:"password"`
	ClientIDPrefix string `yaml:"client_id_prefix"`
	TopicFormat    string `yaml:"topic_format"`
	QoS            byte   `yaml:"qos"`
	CleanSession   bool   `yaml:"clean_session"`
}

// TCPScenarioConfig holds TCP-specific scenario config
type TCPScenarioConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"` // client or server
}

// ModbusScenarioConfig holds Modbus-specific scenario config
type ModbusScenarioConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Mode     string `yaml:"mode"` // master or slave
	SlaveIDs []byte `yaml:"slave_ids"`
}

// OPCUAScenarioConfig holds OPC UA-specific scenario config
type OPCUAScenarioConfig struct {
	Endpoint    string   `yaml:"endpoint"`
	PollInterval string  `yaml:"poll_interval"`
	NodeIDs     []string `yaml:"node_ids"`
}

// RuntimeConfig holds runtime settings
type RuntimeConfig struct {
	Duration  string `yaml:"duration"`
	StartDelay string `yaml:"start_delay"`
	Stagger   string `yaml:"stagger"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// LoadGlobalConfig loads the global configuration from file
func LoadGlobalConfig(path string) (*GlobalConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config SimulatorConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config.Simulator, nil
}

// LoadScenarioConfig loads a scenario configuration from file
func LoadScenarioConfig(path string) (*ScenarioConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read scenario file: %w", err)
	}

	var config ScenarioConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse scenario file: %w", err)
	}

	return &config, nil
}

// LoadDeviceTemplates loads device templates from a directory
func LoadDeviceTemplates(dir string) (map[string]*devices.DeviceTemplateLoaded, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read devices directory: %w", err)
	}

	templates := make(map[string]*devices.DeviceTemplateLoaded)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := fmt.Sprintf("%s/%s", dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read template %s: %w", path, err)
		}

		var templateCfg struct {
			Device devices.DeviceTemplate `yaml:"device_template"`
		}
		if err := yaml.Unmarshal(data, &templateCfg); err != nil {
			return nil, fmt.Errorf("failed to parse template %s: %w", path, err)
		}

		loaded, err := devices.LoadDeviceTemplate(templateCfg.Device)
		if err != nil {
			return nil, fmt.Errorf("failed to load template %s: %w", path, err)
		}

		// Use filename without extension as key
		name := entry.Name()
		if len(name) > 5 && name[len(name)-5:] == ".yaml" {
			name = name[:len(name)-5]
		}
		templates[name] = loaded
	}

	return templates, nil
}

// GetDuration parses a duration string with default value
func GetDuration(s string, defaultVal time.Duration) time.Duration {
	if s == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultVal
	}
	return d
}
