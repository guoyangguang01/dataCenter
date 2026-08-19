package rule

import (
	"context"
	"fmt"
	"sync"

	"github.com/datacenter/internal/message"
)

// Rule 规则定义
type Rule struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	DomainID string            `json:"domain_id"`
	Topic    string            `json:"topic"` // NATS subject to subscribe
	Chain    []NodeConfig      `json:"chain"`
	Enabled  bool              `json:"enabled"`
}

// NodeConfig 节点配置
type NodeConfig struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

// Node 规则节点接口
type Node interface {
	ID() string
	Type() string
	Execute(ctx context.Context, env *message.DeviceEnvelope, state *PipelineState) (*message.DeviceEnvelope, error)
}

// Pipeline 规则链执行器
type Pipeline struct {
	rule  Rule
	nodes []Node
}

// NewPipeline 创建规则链
func NewPipeline(rule Rule, registry *Registry) (*Pipeline, error) {
	nodes := make([]Node, 0, len(rule.Chain))
	for _, cfg := range rule.Chain {
		node, err := registry.Create(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create node %s: %w", cfg.ID, err)
		}
		nodes = append(nodes, node)
	}
	return &Pipeline{rule: rule, nodes: nodes}, nil
}

// Execute 执行规则链
func (p *Pipeline) Execute(ctx context.Context, env *message.DeviceEnvelope) (*message.DeviceEnvelope, error) {
	state := NewPipelineState()
	current := env

	for _, node := range p.nodes {
		var err error
		current, err = node.Execute(ctx, current, state)
		if err != nil {
			return nil, fmt.Errorf("node %s failed: %w", node.ID(), err)
		}
		if current == nil {
			return nil, nil // 消息被过滤
		}
	}
	return current, nil
}

// Registry 节点注册中心
type Registry struct {
	factories map[string]NodeFactory
}

// NodeFactory 节点工厂函数
type NodeFactory func(id string, config map[string]interface{}) (Node, error)

// NewRegistry 创建节点注册中心
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]NodeFactory),
	}
}

// Register 注册节点类型
func (r *Registry) Register(nodeType string, factory NodeFactory) {
	r.factories[nodeType] = factory
}

// Create 创建节点
func (r *Registry) Create(cfg NodeConfig) (Node, error) {
	factory, ok := r.factories[cfg.Type]
	if !ok {
		return nil, fmt.Errorf("unknown node type: %s", cfg.Type)
	}
	return factory(cfg.ID, cfg.Config)
}

// Engine 规则引擎
type Engine struct {
	rules    map[string]*Rule
	pipelines map[string]*Pipeline
	registry *Registry
	mu       sync.RWMutex
}

// NewEngine 创建规则引擎
func NewEngine(registry *Registry) *Engine {
	return &Engine{
		rules:     make(map[string]*Rule),
		pipelines: make(map[string]*Pipeline),
		registry:  registry,
	}
}

// AddRule 添加规则
func (e *Engine) AddRule(rule Rule) error {
	pipeline, err := NewPipeline(rule, e.registry)
	if err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules[rule.ID] = &rule
	e.pipelines[rule.ID] = pipeline
	fmt.Println("[Rule] rule added", rule.ID, rule.Name)
	return nil
}

// RemoveRule 移除规则
func (e *Engine) RemoveRule(ruleID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.rules, ruleID)
	delete(e.pipelines, ruleID)
	fmt.Println("[Rule] rule removed", ruleID)
}

// Process 处理消息
func (e *Engine) Process(ctx context.Context, env *message.DeviceEnvelope) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for id, pipeline := range e.pipelines {
		result, err := pipeline.Execute(ctx, env)
		if err != nil {
			fmt.Println("[Rule] pipeline error:", id, err)
			continue
		}
		if result != nil {
			fmt.Println("[Rule] pipeline produced output:", id)
		}
	}
	return nil
}
