# 测试数据模拟器实现总结

> 完成日期: 2026-08-20 | 状态: 实现完成

## 实现概览

已成功实现 IoT 数据中台的测试数据模拟器，支持 4 种协议的完整模拟。

## 实现的组件

### 1. 核心框架

| 文件 | 说明 |
|------|------|
| `internal/simulator/config.go` | 配置加载系统 |
| `internal/simulator/engine.go` | 模拟引擎核心 |
| `internal/simulator/devices/patterns.go` | 数据生成模式 |
| `internal/simulator/devices/template.go` | 设备模板定义 |
| `cmd/simulator/main.go` | 模拟器入口 |

### 2. 协议适配器

| 协议 | 文件 | 功能 |
|------|------|------|
| MQTT | `internal/simulator/mqtt/adapter.go` | 连接 MQTT Broker 发布数据 |
| TCP | `internal/simulator/tcp/adapter.go` | 支持客户端/服务端双向模拟 |
| Modbus | `internal/simulator/modbus/adapter.go` | 支持主站/从站双向模拟 |
| OPC UA | `internal/simulator/opcua/adapter.go` | 内嵌 OPC UA 节点模拟 |

### 3. 配置文件

```
configs/simulator/
├── simulator.yaml              # 全局配置
├── devices/
│   ├── thermometer.yaml        # 温湿度传感器模板
│   ├── motor.yaml              # 工业电机模板
│   ├── power_meter.yaml        # 智能电表模板
│   └── gps_tracker.yaml        # GPS定位追踪器模板
└── scenarios/
    ├── mqtt_sim.yaml           # MQTT 模拟场景
    ├── tcp_sim.yaml            # TCP 模拟场景
    ├── modbus_sim.yaml         # Modbus 模拟场景
    └── opcua_sim.yaml          # OPC UA 模拟场景
```

### 4. 单元测试

| 测试文件 | 测试内容 |
|----------|----------|
| `internal/simulator/devices/patterns_test.go` | 所有数据生成模式 |

## 数据生成模式

| 模式 | 说明 | 应用场景 |
|------|------|----------|
| `sine` | 正弦波 | 温度、湿度等周期性变化 |
| `random_walk` | 随机游走 | 流量、电流等波动性数据 |
| `step` | 阶梯变化 | 开关状态、档位 |
| `pulse` | 脉冲/突发 | 报警、事件触发 |
| `constant_noise` | 常量+噪声 | 基准值、电池电量 |

## 使用方法

### 启动模拟器

```bash
# MQTT 模拟器
make sim-mqtt

# TCP 客户端模拟器
make sim-tcp-client

# TCP 服务端模拟器
make sim-tcp-server

# Modbus 从站模拟器
make sim-modbus-slave

# Modbus 主站模拟器
make sim-modbus-master

# OPC UA 模拟器
make sim-opcua
```

### 命令行参数

```bash
go run cmd/simulator/main.go \
  --protocol mqtt \
  --config configs/simulator/scenarios/mqtt_sim.yaml \
  --mode client \
  --verbose
```

| 参数 | 说明 |
|------|------|
| `--protocol` | 协议类型 (mqtt/tcp/modbus/opcua) |
| `--config` | 场景配置文件路径 |
| `--mode` | 模式 (client/server, master/slave) |
| `--verbose` | 启用详细日志 |

## 测试结果

```
=== RUN   TestSinePattern
--- PASS: TestSinePattern (0.00s)
=== RUN   TestSinePatternWithNoise
--- PASS: TestSinePatternWithNoise (0.00s)
=== RUN   TestRandomWalkPattern
--- PASS: TestRandomWalkPattern (0.00s)
=== RUN   TestStepPattern
--- PASS: TestStepPattern (0.00s)
=== RUN   TestPulsePattern
--- PASS: TestPulsePattern (0.00s)
=== RUN   TestConstantNoisePattern
--- PASS: TestConstantNoisePattern (0.00s)
=== RUN   TestNewPattern
--- PASS: TestNewPattern (0.00s)
PASS
ok  	github.com/datacenter/internal/simulator/devices	0.037s
```

## 架构特点

1. **协议层模拟** - 真实实现协议栈，测试网关完整处理链路
2. **配置驱动** - YAML 定义设备模板和场景，灵活可扩展
3. **独立进程** - 不污染网关代码，独立启动/停止
4. **双向模拟** - TCP 和 Modbus 支持客户端/服务端、主站/从站双向模拟
5. **结构化日志** - 使用 zerolog 输出 JSON 格式日志
6. **断线退出** - 不重连，断线后直接退出

## 后续扩展

- [ ] 添加 HTTP API 运行时控制
- [ ] 添加更多设备模板
- [ ] 支持历史数据回放
- [ ] 支持加速模式
- [ ] 添加集成测试
