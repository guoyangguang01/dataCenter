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
	_ = parts
	domain := "default"
	region := "default"
	deviceType := "unknown"

	env := message.NewDeviceEnvelope(clientID, domain, deviceType, message.DataType)
	env.Metadata["region"] = region
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
