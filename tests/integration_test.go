package tests

import (
	"testing"
	"github.com/datacenter/internal/message"
	"github.com/datacenter/internal/rule"
	"github.com/datacenter/internal/model"
	"github.com/datacenter/pkg/nats"
)

func TestMessageEnvelope(t *testing.T) {
	env := message.NewDeviceEnvelope("sensor-001", "factory-a", "temp_sensor", message.DataType)
	env.AddUnit("property/temperature", []byte("25.6"))
	if env.DeviceID != "sensor-001" {
		t.Errorf("expected sensor-001, got %s", env.DeviceID)
	}
}

func TestTopicWildcard(t *testing.T) {
	if !nats.MatchWildcard("domains.>.devices.>.>.>.up", "domains.factory-a.devices.cn-east.sensor.temp-001.up") {
		t.Error("expected match")
	}
}

func TestRuleEngine(t *testing.T) {
	registry := rule.NewRegistry()
	registry.Register("filter", rule.NewFilterNode)
	r := rule.Rule{ID: "test", Chain: []rule.NodeConfig{{ID: "f1", Type: "filter", Config: map[string]interface{}{"field": "topic", "operator": "contains", "value": "temperature"}}}}
	p, _ := rule.NewPipeline(r, registry)
	env := message.NewDeviceEnvelope("s1", "d1", "t1", message.DataType)
	env.AddUnit("property/temperature", []byte("25.6"))
	result, _ := p.Execute(nil, env)
	if result == nil {
		t.Error("expected result")
	}
}

func TestModel(t *testing.T) {
	m := model.ThingModel{ID: "v1", Properties: []model.PropertyDef{{ID: "temp", Range: [2]float64{-40, 125}}}}
	if len(m.Properties) != 1 {
		t.Error("expected 1 property")
	}
}
