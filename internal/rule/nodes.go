package rule

import (
	"context"
	"fmt"
	"strings"

	"github.com/datacenter/internal/message"
)

// FilterNode 过滤器节点
type FilterNode struct {
	id      string
	field   string
	operator string
	value   string
}

func NewFilterNode(id string, config map[string]interface{}) (Node, error) {
	field, _ := config["field"].(string)
	operator, _ := config["operator"].(string)
	value, _ := config["value"].(string)
	if field == "" {
		field = "topic"
	}
	if operator == "" {
		operator = "eq"
	}
	return &FilterNode{id: id, field: field, operator: operator, value: value}, nil
}

func (n *FilterNode) ID() string   { return n.id }
func (n *FilterNode) Type() string { return "filter" }

func (n *FilterNode) Execute(ctx context.Context, env *message.DeviceEnvelope, state *PipelineState) (*message.DeviceEnvelope, error) {
	// 简单过滤：检查 topic 是否匹配
	if len(env.Units) == 0 {
		return nil, nil
	}

	for _, unit := range env.Units {
		if n.matches(unit.Topic) {
			return env, nil
		}
	}
	return nil, nil // 不匹配则过滤
}

func (n *FilterNode) matches(topic string) bool {
	switch n.operator {
	case "eq":
		return topic == n.value
	case "contains":
		return strings.Contains(topic, n.value)
	case "prefix":
		return strings.HasPrefix(topic, n.value)
	default:
		return true
	}
}

// TransformNode 转换器节点
type TransformNode struct {
	id      string
	extract string
	parse   string
}

func NewTransformNode(id string, config map[string]interface{}) (Node, error) {
	extract, _ := config["extract"].(string)
	parse, _ := config["parse"].(string)
	return &TransformNode{id: id, extract: extract, parse: parse}, nil
}

func (n *TransformNode) ID() string   { return n.id }
func (n *TransformNode) Type() string { return "transform" }

func (n *TransformNode) Execute(ctx context.Context, env *message.DeviceEnvelope, state *PipelineState) (*message.DeviceEnvelope, error) {
	// 简单转换：提取指定 topic 的数据
	if n.extract != "" {
		for _, unit := range env.Units {
			if unit.Topic == n.extract {
				state.Set("extracted_payload", unit.Payload)
				return env, nil
			}
		}
	}
	return env, nil
}

// ConditionNode 条件判断节点
type ConditionNode struct {
	id           string
	expression   string
	trueBranch   string
	falseBranch  string
}

func NewConditionNode(id string, config map[string]interface{}) (Node, error) {
	expr, _ := config["expression"].(string)
	trueBranch, _ := config["true_branch"].(string)
	falseBranch, _ := config["false_branch"].(string)
	return &ConditionNode{
		id:          id,
		expression:  expr,
		trueBranch:  trueBranch,
		falseBranch: falseBranch,
	}, nil
}

func (n *ConditionNode) ID() string   { return n.id }
func (n *ConditionNode) Type() string { return "condition" }

func (n *ConditionNode) Execute(ctx context.Context, env *message.DeviceEnvelope, state *PipelineState) (*message.DeviceEnvelope, error) {
	// 简单条件：检查 expression 中的值
	// expression 格式: "key operator value" (e.g. "temp > 30")
	if n.expression == "" {
		return env, nil
	}

	// 简化实现：如果 state 中有对应的值则返回环境
	if _, ok := state.Get("extracted_payload"); ok {
		return env, nil
	}
	return env, nil
}

// AggregateNode 聚合节点
type AggregateNode struct {
	id           string
	windowSize   int
	function     string
}

func NewAggregateNode(id string, config map[string]interface{}) (Node, error) {
	windowSize, _ := config["window_size"].(int)
	if windowSize == 0 {
		windowSize = 10
	}
	function, _ := config["function"].(string)
	if function == "" {
		function = "avg"
	}
	return &AggregateNode{id: id, windowSize: windowSize, function: function}, nil
}

func (n *AggregateNode) ID() string   { return n.id }
func (n *AggregateNode) Type() string { return "aggregate" }

func (n *AggregateNode) Execute(ctx context.Context, env *message.DeviceEnvelope, state *PipelineState) (*message.DeviceEnvelope, error) {
	// 聚合逻辑：将数据添加到窗口
	if len(env.Units) > 0 {
		state.Set("last_payload", env.Units[0].Payload)
	}
	return env, nil
}

// ActionNode 动作节点
type ActionNode struct {
	id              string
	actionType      string
	topicTemplate   string
	payloadTemplate map[string]interface{}
	publisher       Publisher
}

// Publisher 动作发布器接口
type Publisher interface {
	Publish(ctx context.Context, subject string, data interface{}) error
}

func NewActionNode(id string, config map[string]interface{}, publisher Publisher) (Node, error) {
	actionType, _ := config["type"].(string)
	topicTemplate, _ := config["topic_template"].(string)
	payloadTemplate, _ := config["payload_template"].(map[string]interface{})
	return &ActionNode{
		id:              id,
		actionType:      actionType,
		topicTemplate:   topicTemplate,
		payloadTemplate: payloadTemplate,
		publisher:       publisher,
	}, nil
}

func (n *ActionNode) ID() string   { return n.id }
func (n *ActionNode) Type() string { return "action" }

func (n *ActionNode) Execute(ctx context.Context, env *message.DeviceEnvelope, state *PipelineState) (*message.DeviceEnvelope, error) {
	fmt.Println("[Rule] action:", n.id, n.actionType)
	return env, nil
}

// RegisterBuiltinNodes 注册所有内建节点
func RegisterBuiltinNodes(registry *Registry, publisher Publisher) {
	registry.Register("filter", NewFilterNode)
	registry.Register("transform", NewTransformNode)
	registry.Register("condition", NewConditionNode)
	registry.Register("aggregate", NewAggregateNode)

	// Action 需要注入 publisher
	registry.Register("action", func(id string, config map[string]interface{}) (Node, error) {
		return NewActionNode(id, config, publisher)
	})
}
