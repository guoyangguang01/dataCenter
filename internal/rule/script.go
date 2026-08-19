package rule

import (
	"context"
	"fmt"
	"time"

	"github.com/datacenter/internal/message"
	"github.com/dop251/goja"
)

// ScriptNode JavaScript 脚本节点
type ScriptNode struct {
	id       string
	script   string
	timeout  time.Duration
}

func NewScriptNode(id string, config map[string]interface{}) (Node, error) {
	script, _ := config["script"].(string)
	timeoutMs, _ := config["timeout_ms"].(int)
	if timeoutMs == 0 {
		timeoutMs = 5000 // 默认 5 秒
	}
	return &ScriptNode{
		id:      id,
		script:  script,
		timeout: time.Duration(timeoutMs) * time.Millisecond,
	}, nil
}

func (n *ScriptNode) ID() string   { return n.id }
func (n *ScriptNode) Type() string { return "script" }

func (n *ScriptNode) Execute(ctx context.Context, env *message.DeviceEnvelope, state *PipelineState) (*message.DeviceEnvelope, error) {
	vm := goja.New()

	// 注入数据到 JS 环境
	vm.Set("device_id", env.DeviceID)
	vm.Set("domain_id", env.DomainID)
	vm.Set("timestamp", env.Timestamp)

	// 注入 state
	stateData := make(map[string]interface{})
	for _, unit := range env.Units {
		stateData[unit.Topic] = string(unit.Payload)
	}
	vm.Set("data", stateData)

	// 注入辅助函数
	vm.Set("log", func(call goja.FunctionCall) goja.Value {
		fmt.Println("[Script]", call.Arguments[0].Export())
		return goja.Undefined()
	})

	// 执行脚本（带超时）
	done := make(chan error, 1)
	go func() {
		_, err := vm.RunString(n.script)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("script execution failed: %w", err)
		}
	case <-time.After(n.timeout):
		return nil, fmt.Errorf("script execution timeout")
	}

	return env, nil
}

// RegisterScriptNode 注册脚本节点
func RegisterScriptNode(registry *Registry) {
	registry.Register("script", NewScriptNode)
}
