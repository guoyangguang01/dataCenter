package mqtt

import (
	"testing"

	"github.com/datacenter/internal/message"
)

func TestCodec_ToEnvelope(t *testing.T) {
	codec := NewCodec()

	tests := []struct {
		name       string
		clientID   string
		topic      string
		wantDomain string
		wantRegion string
		wantType   string
	}{
		{
			name:       "full topic",
			clientID:   "sensor-001",
			topic:      "devices/factory-a/east/motor-01/telemetry",
			wantDomain: "factory-a",
			wantRegion: "east",
			wantType:   "actuator",
		},
		{
			name:       "meter device",
			clientID:   "meter-001",
			topic:      "devices/factory-b/west/meter-001/telemetry",
			wantDomain: "factory-b",
			wantRegion: "west",
			wantType:   "sensor",
		},
		{
			name:       "short topic defaults",
			clientID:   "dev-001",
			topic:      "devices/factory-a",
			wantDomain: "default",  // only 2 parts, doesn't meet >= 3 threshold
			wantRegion: "default",
			wantType:   "sensor",
		},
		{
			name:       "minimal topic",
			clientID:   "dev-001",
			topic:      "data",
			wantDomain: "default",
			wantRegion: "default",
			wantType:   "sensor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := codec.ToEnvelope(tt.clientID, tt.topic, []byte(`{"value":1}`))

			if env.DeviceID != tt.clientID {
				t.Errorf("DeviceID: got %s, want %s", env.DeviceID, tt.clientID)
			}
			if env.DomainID != tt.wantDomain {
				t.Errorf("DomainID: got %s, want %s", env.DomainID, tt.wantDomain)
			}
			if env.Metadata["region"] != tt.wantRegion {
				t.Errorf("region: got %s, want %s", env.Metadata["region"], tt.wantRegion)
			}
			if env.Metadata["device_type"] != tt.wantType {
				t.Errorf("device_type: got %s, want %s", env.Metadata["device_type"], tt.wantType)
			}
			if env.Type != message.DataType {
				t.Errorf("Type: got %d, want %d", env.Type, message.DataType)
			}
			if len(env.Units) != 1 {
				t.Fatalf("expected 1 unit, got %d", len(env.Units))
			}
			if env.Units[0].Topic != tt.topic {
				t.Errorf("Unit topic: got %s, want %s", env.Units[0].Topic, tt.topic)
			}
		})
	}
}

func TestCodec_ToMQTTTopic(t *testing.T) {
	codec := NewCodec()

	env := message.NewDeviceEnvelope("d1", "domain", "model", message.DataType)
	env.AddUnit("devices/factory/temp", []byte(`{}`))

	topic := codec.ToMQTTTopic(env)
	if topic != "devices/factory/temp" {
		t.Errorf("got %s, want devices/factory/temp", topic)
	}
}

func TestCodec_ToMQTTTopic_EmptyUnits(t *testing.T) {
	codec := NewCodec()

	env := message.NewDeviceEnvelope("d1", "domain", "model", message.DataType)

	topic := codec.ToMQTTTopic(env)
	if topic != "" {
		t.Errorf("expected empty string for no units, got %s", topic)
	}
}

func TestCodec_ToMQTTPayload(t *testing.T) {
	codec := NewCodec()

	env := message.NewDeviceEnvelope("d1", "domain", "model", message.DataType)
	env.AddUnit("topic", []byte(`{}`))

	payload := codec.ToMQTTPayload(env)
	if len(payload) == 0 {
		t.Errorf("expected non-empty payload")
	}
}

func TestCodec_ToMQTTPayload_EmptyUnits(t *testing.T) {
	codec := NewCodec()

	env := message.NewDeviceEnvelope("d1", "domain", "model", message.DataType)

	payload := codec.ToMQTTPayload(env)
	if payload != nil {
		t.Errorf("expected nil for empty units, got %v", payload)
	}
}
