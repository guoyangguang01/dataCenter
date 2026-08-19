package message

import (
	"encoding/json"
	"testing"
)

func TestDeviceEnvelope(t *testing.T) {
	env := NewDeviceEnvelope("sensor-001", "factory-a", "temp_sensor_v1", DataType)
	env.AddUnit("property/temperature", []byte(`{"value":25.6}`))

	if env.DeviceID != "sensor-001" {
		t.Errorf("expected device_id sensor-001, got %s", env.DeviceID)
	}
	if len(env.Units) != 1 {
		t.Errorf("expected 1 unit, got %d", len(env.Units))
	}
	if env.Units[0].Topic != "property/temperature" {
		t.Errorf("expected topic property/temperature, got %s", env.Units[0].Topic)
	}
}

func TestDeviceEnvelopeJSON(t *testing.T) {
	env := NewDeviceEnvelope("sensor-001", "factory-a", "temp_sensor_v1", DataType)
	env.AddUnit("property/temperature", []byte(`{"value":25.6}`))

	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded DeviceEnvelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.DeviceID != "sensor-001" {
		t.Errorf("expected device_id sensor-001, got %s", decoded.DeviceID)
	}
}

func TestBatchUnits(t *testing.T) {
	env := NewDeviceEnvelope("gateway-01", "factory-a", "gateway_v1", DataType)
	env.
		AddUnit("property/temperature", []byte(`{"value":25.6}`)).
		AddUnit("property/humidity", []byte(`{"value":68}`)).
		AddUnit("property/pressure", []byte(`{"value":1013}`))

	if len(env.Units) != 3 {
		t.Errorf("expected 3 units, got %d", len(env.Units))
	}
}
