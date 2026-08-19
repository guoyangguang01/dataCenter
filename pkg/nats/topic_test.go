package nats

import "testing"

func TestTopicBuilder(t *testing.T) {
	topic := NewTopicBuilder().
		Domains("factory-a").
		Devices().
		Region("cn-east").
		DeviceType("sensor").
		DeviceID("temp-001").
		Direction(DirectionUp).
		Build()
	expected := "domains.factory-a.devices.cn-east.sensor.temp-001.up"
	if topic != expected {
		t.Errorf("expected %s, got %s", expected, topic)
	}
}

func TestDeviceReportTopic(t *testing.T) {
	topic := DeviceReportTopic("factory-a", "cn-east", "sensor", "temp-001")
	expected := "domains.factory-a.devices.cn-east.sensor.temp-001.up"
	if topic != expected {
		t.Errorf("expected %s, got %s", expected, topic)
	}
}

func TestDomainAllReportTopic(t *testing.T) {
	topic := DomainAllReportTopic("factory-a")
	expected := "domains.factory-a.devices.>.>.>.up"
	if topic != expected {
		t.Errorf("expected %s, got %s", expected, topic)
	}
}

func TestAllSensorReportTopic(t *testing.T) {
	topic := AllSensorReportTopic()
	expected := "domains.>.devices.>.sensor.>.up"
	if topic != expected {
		t.Errorf("expected %s, got %s", expected, topic)
	}
}

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		pattern string
		topic   string
		match   bool
	}{
		{"domains.>.devices.>.>.>.up", "domains.factory-a.devices.cn-east.sensor.temp-001.up", true},
		{"domains.>.devices.>.>.>.up", "domains.factory-a.devices.cn-east.controller.ac-01.up", true},
		{"domains.>.devices.>.sensor.>.up", "domains.factory-a.devices.cn-east.sensor.temp-001.up", true},
		{"domains.>.devices.>.sensor.>.up", "domains.factory-a.devices.cn-east.controller.ac-01.up", false},
		{"domains.factory-a.devices.>.>.>.up", "domains.factory-b.devices.cn-east.sensor.temp-001.up", false},
	}
	for _, tt := range tests {
		result := MatchWildcard(tt.pattern, tt.topic)
		if result != tt.match {
			t.Errorf("pattern=%s topic=%s expected=%v got=%v", tt.pattern, tt.topic, tt.match, result)
		}
	}
}
