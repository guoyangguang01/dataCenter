package rule

import (
	"context"
	"testing"

	"github.com/datacenter/internal/message"
)

// --- Registry 测试 ---

func TestRegistry_RegisterAndCreate(t *testing.T) {
	reg := NewRegistry()
	reg.Register("filter", NewFilterNode)

	cfg := NodeConfig{
		ID:   "f1",
		Type: "filter",
		Config: map[string]interface{}{
			"field":    "topic",
			"operator": "eq",
			"value":    "test/topic",
		},
	}

	node, err := reg.Create(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.ID() != "f1" {
		t.Errorf("expected ID f1, got %s", node.ID())
	}
	if node.Type() != "filter" {
		t.Errorf("expected type filter, got %s", node.Type())
	}
}

func TestRegistry_UnknownType(t *testing.T) {
	reg := NewRegistry()
	cfg := NodeConfig{ID: "x", Type: "nonexistent", Config: nil}

	_, err := reg.Create(cfg)
	if err == nil {
		t.Errorf("expected error for unknown type")
	}
}

// --- FilterNode 测试 ---

func TestFilterNode_Eq(t *testing.T) {
	node, _ := NewFilterNode("f1", map[string]interface{}{
		"operator": "eq",
		"value":    "sensor/temp",
	})

	env := message.NewDeviceEnvelope("d1", "domain", "model", message.DataType)
	env.AddUnit("sensor/temp", []byte(`{"v":1}`))

	result, err := node.Execute(context.Background(), env, NewPipelineState())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Errorf("expected envelope to pass filter")
	}
}

func TestFilterNode_EqNoMatch(t *testing.T) {
	node, _ := NewFilterNode("f1", map[string]interface{}{
		"operator": "eq",
		"value":    "sensor/temp",
	})

	env := message.NewDeviceEnvelope("d1", "domain", "model", message.DataType)
	env.AddUnit("sensor/humidity", []byte(`{"v":1}`))

	result, err := node.Execute(context.Background(), env, NewPipelineState())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil (filtered out)")
	}
}

func TestFilterNode_Contains(t *testing.T) {
	node, _ := NewFilterNode("f1", map[string]interface{}{
		"operator": "contains",
		"value":    "temp",
	})

	env := message.NewDeviceEnvelope("d1", "domain", "model", message.DataType)
	env.AddUnit("sensor/temp/room1", []byte(`{}`))

	result, _ := node.Execute(context.Background(), env, NewPipelineState())
	if result == nil {
		t.Errorf("expected match for contains")
	}
}

func TestFilterNode_Prefix(t *testing.T) {
	node, _ := NewFilterNode("f1", map[string]interface{}{
		"operator": "prefix",
		"value":    "sensor/",
	})

	env := message.NewDeviceEnvelope("d1", "domain", "model", message.DataType)
	env.AddUnit("sensor/temp", []byte(`{}`))

	result, _ := node.Execute(context.Background(), env, NewPipelineState())
	if result == nil {
		t.Errorf("expected match for prefix")
	}
}

func TestFilterNode_EmptyUnits(t *testing.T) {
	node, _ := NewFilterNode("f1", map[string]interface{}{
		"operator": "eq",
		"value":    "x",
	})

	env := message.NewDeviceEnvelope("d1", "domain", "model", message.DataType)
	// 不添加 unit

	result, _ := node.Execute(context.Background(), env, NewPipelineState())
	if result != nil {
		t.Errorf("expected nil for empty units")
	}
}

func TestFilterNode_DefaultOperator(t *testing.T) {
	// 空 operator 默认为 eq
	node, _ := NewFilterNode("f1", map[string]interface{}{
		"value": "match/me",
	})

	env := message.NewDeviceEnvelope("d1", "domain", "model", message.DataType)
	env.AddUnit("match/me", []byte(`{}`))

	result, _ := node.Execute(context.Background(), env, NewPipelineState())
	if result == nil {
		t.Errorf("expected match with default eq operator")
	}
}

// --- TransformNode 测试 ---

func TestTransformNode_Extract(t *testing.T) {
	node, _ := NewTransformNode("t1", map[string]interface{}{
		"extract": "sensor/temp",
	})

	env := message.NewDeviceEnvelope("d1", "domain", "model", message.DataType)
	env.AddUnit("sensor/temp", []byte(`{"value":25}`))
	env.AddUnit("sensor/humidity", []byte(`{"value":60}`))

	state := NewPipelineState()
	result, err := node.Execute(context.Background(), env, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	extracted, ok := state.Get("extracted_payload")
	if !ok {
		t.Fatal("expected extracted_payload in state")
	}
	if string(extracted.([]byte)) != `{"value":25}` {
		t.Errorf("unexpected extracted payload: %v", extracted)
	}
}

func TestTransformNode_ExtractNoMatch(t *testing.T) {
	node, _ := NewTransformNode("t1", map[string]interface{}{
		"extract": "sensor/pressure",
	})

	env := message.NewDeviceEnvelope("d1", "domain", "model", message.DataType)
	env.AddUnit("sensor/temp", []byte(`{}`))

	state := NewPipelineState()
	result, _ := node.Execute(context.Background(), env, state)
	// 不匹配时仍然返回 env
	if result == nil {
		t.Errorf("expected env returned even when no match")
	}
	_, ok := state.Get("extracted_payload")
	if ok {
		t.Errorf("expected no extracted_payload when no match")
	}
}

// --- ConditionNode 测试 ---

func TestConditionNode_EmptyExpression(t *testing.T) {
	node, _ := NewConditionNode("c1", map[string]interface{}{})

	env := message.NewDeviceEnvelope("d1", "domain", "model", message.DataType)
	result, _ := node.Execute(context.Background(), env, NewPipelineState())
	if result == nil {
		t.Errorf("expected env returned for empty expression")
	}
}

func TestConditionNode_WithExtractedPayload(t *testing.T) {
	node, _ := NewConditionNode("c1", map[string]interface{}{
		"expression": "temp > 30",
	})

	env := message.NewDeviceEnvelope("d1", "domain", "model", message.DataType)
	state := NewPipelineState()
	state.Set("extracted_payload", []byte(`{"value":35}`))

	result, _ := node.Execute(context.Background(), env, state)
	if result == nil {
		t.Errorf("expected env returned when state has extracted_payload")
	}
}

// --- AggregateNode 测试 ---

func TestAggregateNode_DefaultConfig(t *testing.T) {
	node, _ := NewAggregateNode("a1", map[string]interface{}{})

	if node.(*AggregateNode).windowSize != 10 {
		t.Errorf("expected default windowSize 10, got %d", node.(*AggregateNode).windowSize)
	}
	if node.(*AggregateNode).function != "avg" {
		t.Errorf("expected default function avg, got %s", node.(*AggregateNode).function)
	}
}

func TestAggregateNode_Execute(t *testing.T) {
	node, _ := NewAggregateNode("a1", map[string]interface{}{
		"window_size": 5,
		"function":    "sum",
	})

	env := message.NewDeviceEnvelope("d1", "domain", "model", message.DataType)
	env.AddUnit("sensor/temp", []byte(`{"value":25}`))

	state := NewPipelineState()
	result, err := node.Execute(context.Background(), env, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	lastPayload, ok := state.Get("last_payload")
	if !ok {
		t.Fatal("expected last_payload in state")
	}
	if string(lastPayload.([]byte)) != `{"value":25}` {
		t.Errorf("unexpected last_payload: %v", lastPayload)
	}
}

// --- Pipeline 测试 ---

func TestPipeline_Execute(t *testing.T) {
	reg := NewRegistry()
	RegisterBuiltinNodes(reg, nil)

	rule := Rule{
		ID:   "r1",
		Name: "test-rule",
		Chain: []NodeConfig{
			{
				ID:   "f1",
				Type: "filter",
				Config: map[string]interface{}{
					"operator": "prefix",
					"value":    "sensor/",
				},
			},
			{
				ID:   "t1",
				Type: "transform",
				Config: map[string]interface{}{
					"extract": "sensor/temp",
				},
			},
		},
	}

	pipeline, err := NewPipeline(rule, reg)
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	// 匹配的消息
	env := message.NewDeviceEnvelope("d1", "domain", "model", message.DataType)
	env.AddUnit("sensor/temp", []byte(`{"value":25}`))

	result, err := pipeline.Execute(context.Background(), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// 不匹配的消息应该被过滤
	env2 := message.NewDeviceEnvelope("d1", "domain", "model", message.DataType)
	env2.AddUnit("actuator/motor", []byte(`{}`))

	result2, err := pipeline.Execute(context.Background(), env2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2 != nil {
		t.Errorf("expected nil (filtered)")
	}
}

func TestPipeline_UnknownNodeType(t *testing.T) {
	reg := NewRegistry()

	rule := Rule{
		ID:    "r1",
		Chain: []NodeConfig{{ID: "x", Type: "nonexistent", Config: nil}},
	}

	_, err := NewPipeline(rule, reg)
	if err == nil {
		t.Errorf("expected error for unknown node type")
	}
}

// --- Engine 测试 ---

func TestEngine_AddRemoveRule(t *testing.T) {
	reg := NewRegistry()
	RegisterBuiltinNodes(reg, nil)

	engine := NewEngine(reg)

	rule := Rule{
		ID:   "r1",
		Name: "test",
		Chain: []NodeConfig{
			{ID: "f1", Type: "filter", Config: map[string]interface{}{"value": "x"}},
		},
	}

	if err := engine.AddRule(rule); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	// 处理消息
	env := message.NewDeviceEnvelope("d1", "domain", "model", message.DataType)
	env.AddUnit("x", []byte(`{}`))

	err := engine.Process(context.Background(), env)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// 移除规则
	engine.RemoveRule("r1")
}

func TestEngine_ProcessMultipleRules(t *testing.T) {
	reg := NewRegistry()
	RegisterBuiltinNodes(reg, nil)

	engine := NewEngine(reg)

	engine.AddRule(Rule{
		ID:    "r1",
		Chain: []NodeConfig{{ID: "f1", Type: "filter", Config: map[string]interface{}{"value": "a"}}},
	})
	engine.AddRule(Rule{
		ID:    "r2",
		Chain: []NodeConfig{{ID: "f2", Type: "filter", Config: map[string]interface{}{"value": "b"}}},
	})

	env := message.NewDeviceEnvelope("d1", "domain", "model", message.DataType)
	env.AddUnit("a", []byte(`{}`))

	err := engine.Process(context.Background(), env)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}
}
