package devices

import (
	"fmt"
	"time"
)

// DeviceTemplate defines the structure of a device template
type DeviceTemplate struct {
	Name           string            `yaml:"name"`
	ModelID        string            `yaml:"model_id"`
	DataPoints     []DataPointConfig `yaml:"data_points"`
	ReportInterval string            `yaml:"report_interval"`
	Metadata       map[string]string `yaml:"metadata"`
}

// DataPointConfig defines a data point configuration
type DataPointConfig struct {
	Name        string       `yaml:"name"`
	Unit        string       `yaml:"unit"`
	Description string       `yaml:"description,omitempty"`
	Pattern     PatternConfig `yaml:"pattern"`
	Range       [2]float64   `yaml:"range,omitempty"`
	Precision   int          `yaml:"precision,omitempty"`
}

// DeviceTemplateLoaded is a loaded device template with compiled patterns
type DeviceTemplateLoaded struct {
	Name           string
	ModelID        string
	DataPoints     []DataPointLoaded
	ReportInterval time.Duration
	Metadata       map[string]string
}

// DataPointLoaded is a loaded data point with compiled pattern
type DataPointLoaded struct {
	Name        string
	Unit        string
	Description string
	Pattern     Pattern
	Range       [2]float64
	Precision   int
}

// LoadDeviceTemplate loads a device template from configuration
func LoadDeviceTemplate(cfg DeviceTemplate) (*DeviceTemplateLoaded, error) {
	interval, err := time.ParseDuration(cfg.ReportInterval)
	if err != nil {
		interval = 5 * time.Second
	}

	dataPoints := make([]DataPointLoaded, 0, len(cfg.DataPoints))
	for _, dpCfg := range cfg.DataPoints {
		pattern := NewPattern(dpCfg.Pattern)
		dp := DataPointLoaded{
			Name:        dpCfg.Name,
			Unit:        dpCfg.Unit,
			Description: dpCfg.Description,
			Pattern:     pattern,
			Range:       dpCfg.Range,
			Precision:   dpCfg.Precision,
		}
		dataPoints = append(dataPoints, dp)
	}

	return &DeviceTemplateLoaded{
		Name:           cfg.Name,
		ModelID:        cfg.ModelID,
		DataPoints:     dataPoints,
		ReportInterval: interval,
		Metadata:       cfg.Metadata,
	}, nil
}

// DeviceInstance represents a running device instance
type DeviceInstance struct {
	DeviceID     string
	DomainID     string
	Region       string
	DeviceType   string
	ModelID      string
	Template     *DeviceTemplateLoaded
	Metadata     map[string]string
	ReportInterval time.Duration
}

// NewDeviceInstance creates a new device instance
func NewDeviceInstance(
	deviceID, domainID, region, deviceType string,
	template *DeviceTemplateLoaded,
) *DeviceInstance {
	metadata := make(map[string]string)
	for k, v := range template.Metadata {
		metadata[k] = v
	}
	metadata["region"] = region
	metadata["device_type"] = deviceType

	return &DeviceInstance{
		DeviceID:       deviceID,
		DomainID:       domainID,
		Region:         region,
		DeviceType:     deviceType,
		ModelID:        template.ModelID,
		Template:       template,
		Metadata:       metadata,
		ReportInterval: template.ReportInterval,
	}
}

// GenerateData generates data for all data points at the given time
func (d *DeviceInstance) GenerateData(t time.Time) map[string]interface{} {
	data := make(map[string]interface{})
	for _, dp := range d.Template.DataPoints {
		value := dp.Pattern.Next(t)

		// Apply precision
		if dp.Precision > 0 {
			factor := 1.0
			for i := 0; i < dp.Precision; i++ {
				factor *= 10
			}
			value = float64(int(value*factor)) / factor
		}

		// Clamp to range if specified
		if dp.Range[0] != 0 || dp.Range[1] != 0 {
			if value < dp.Range[0] {
				value = dp.Range[0]
			} else if value > dp.Range[1] {
				value = dp.Range[1]
			}
		}

		data[dp.Name] = value
	}
	return data
}

// GenerateDataWithTimestamp generates data with timestamp as string key
func (d *DeviceInstance) GenerateDataWithTimestamp(t time.Time) map[string]interface{} {
	data := d.GenerateData(t)
	data["timestamp"] = t.UnixMilli()
	data["device_id"] = d.DeviceID
	return data
}

// FormatID formats a device ID with prefix and sequence number
func FormatID(prefix string, index int) string {
	return fmt.Sprintf("%s-%03d", prefix, index+1)
}
