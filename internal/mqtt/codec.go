package mqtt

import (
	"encoding/json"
	"strings"

	"github.com/datacenter/internal/message"
)

type Codec struct{}

func NewCodec() *Codec {
	return &Codec{}
}

func (c *Codec) ToEnvelope(clientID, topic string, payload []byte) *message.DeviceEnvelope {
	parts := strings.Split(topic, "/")
	// topic 格式: devices/{domain_id}/{region}/{device_id}/telemetry
	domain := "default"
	region := "default"
	deviceType := "sensor"

	if len(parts) >= 3 {
		domain = parts[1]
		region = parts[2]
	}
	if len(parts) >= 4 {
		// 尝试从设备ID推断类型
		deviceID := parts[3]
		if strings.HasPrefix(deviceID, "motor") {
			deviceType = "actuator"
		} else if strings.HasPrefix(deviceID, "meter") {
			deviceType = "sensor"
		}
	}

	env := message.NewDeviceEnvelope(clientID, domain, deviceType, message.DataType)
	env.Metadata["region"] = region
	env.Metadata["device_type"] = deviceType
	env.Metadata["topic"] = topic
	env.AddUnit(topic, payload)
	return env
}

func (c *Codec) ToMQTTTopic(env *message.DeviceEnvelope) string {
	if len(env.Units) == 0 {
		return ""
	}
	return env.Units[0].Topic
}

func (c *Codec) ToMQTTPayload(env *message.DeviceEnvelope) []byte {
	if len(env.Units) == 0 {
		return nil
	}
	data, _ := json.Marshal(env)
	return data
}
