# 测试数据模拟器设计文档

> 版本: v1.0 | 日期: 2026-08-20 | 状态: 设计完成

## 1. 概述

### 1.1 目标

为 IoT 数据中台的各个网关（MQTT、TCP、Modbus、OPC UA）添加测试数据模拟器，用于：

- **开发调试**：无真实设备时模拟设备数据流，方便前端和后端开发
- **集成测试**：自动化测试完整数据管道（网关 → NATS → 规则引擎 → 时序写入）

### 1.2 设计原则

| 原则 | 说明 |
|------|------|
| 协议层模拟 | 真实实现协议栈，测试网关完整处理链路 |
| 配置驱动 | YAML 定义设备模板和场景，灵活可扩展 |
| 独立进程 | 不污染网关代码，独立启动/停止 |
| 开箱即用 | 提供完整示例配置，启动即可产生数据 |

---

## 2. 架构设计

### 2.1 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                     Simulator (cmd/simulator)                │
│                                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │   MQTT   │  │   TCP    │  │  Modbus  │  │  OPC UA  │   │
│  │ Simulator│  │ Simulator│  │ Simulator│  │ Simulator│   │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘   │
│       │              │              │              │         │
│  ┌────┴──────────────┴──────────────┴──────────────┴─────┐  │
│  │              Data Generator Engine                     │  │
│  │  (YAML Config → Device Model → Pattern → Payload)     │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
        │              │              │              │
        ▼              ▼              ▼              ▼
   ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐
   │  MQTT   │   │  TCP    │   │  Modbus │   │  OPC UA │
   │  Broker │   │ Gateway │   │ Gateway │   │ Gateway │
   └─────────┘   └─────────┘   └─────────┘   └─────────┘
```

### 2.2 数据流

```
YAML Config → Device Template → Data Generator → Protocol Encoder → Gateway
                                      │
                                      ▼
                               DeviceEnvelope
                                      │
                                      ▼
                               NATS JetStream
                                      │
                                      ▼
                            Rule Engine / Timeseries Writer
```

### 2.3 代码结构

```
internal/simulator/
├── engine.go           # 模拟引擎核心（调度、生命周期管理）
├── generator.go        # 数据生成器（模式：正弦波、随机游走等）
├── config.go           # 配置加载与解析
├── devices/            # 设备模型定义
│   ├── template.go     # 设备模板基类
│   └── patterns.go     # 数据变化模式实现
├── mqtt/               # MQTT 模拟器
│   ├── simulator.go    # MQTT 客户端模拟
│   └── codec.go        # MQTT 协议编解码
├── tcp/                # TCP 模拟器
│   ├── client_sim.go   # TCP 客户端模拟
│   ├── server_sim.go   # TCP 服务端模拟
│   └── frame.go        # TCP 帧编解码
├── modbus/             # Modbus 模拟器
│   ├── master_sim.go   # Modbus 主站模拟
│   ├── slave_sim.go    # Modbus 从站模拟
│   └── frame.go        # Modbus MBAP 帧编解码
└── opcua/              # OPC UA 模拟器
    ├── server.go       # 内嵌 OPC UA 服务器
    └── nodes.go        # 节点定义与数据绑定

cmd/simulator/
└── main.go             # 模拟器入口

configs/simulator/
├── simulator.yaml      # 全局配置
├── devices/            # 设备模板
│   ├── thermometer.yaml
│   ├── motor.yaml
│   ├── power_meter.yaml
│   └── gps_tracker.yaml
└── scenarios/          # 场景配置
    ├── mqtt_sim.yaml
    ├── tcp_sim.yaml
    ├── modbus_sim.yaml
    └── opcua_sim.yaml
```

---

## 3. 核心组件设计

### 3.1 模拟引擎 (engine.go)

```go
type SimulatorEngine struct {
    config     *SimulatorConfig
    protocol   string           // mqtt, tcp, modbus, opcua
    devices    []*DeviceInstance
    generator  *DataGenerator
    adapter    ProtocolAdapter  // 协议适配器接口
    logger     *zerolog.Logger
    quit       chan struct{}
}

type ProtocolAdapter interface {
    Start(ctx context.Context) error
    Stop() error
    SendData(deviceID string, data []byte) error
}
```

**职责**：
- 加载配置，初始化设备实例
- 按配置的间隔调度数据生成
- 管理协议适配器的生命周期
- 断线时直接退出（不重连）

### 3.2 数据生成器 (generator.go)

```go
type DataGenerator struct {
    patterns map[string]Pattern
}

type Pattern interface {
    Next(t time.Time) float64
}

// 支持的模式
type SinePattern struct {       // 正弦波
    Amplitude, Period, Offset, Phase float64
}

type RandomWalkPattern struct {  // 随机游走
    Min, Max, Step float64
    current float64
}

type StepPattern struct {        // 阶梯变化
    Values []float64
    Interval time.Duration
}

type PulsePattern struct {       // 脉冲/突发
    BaseValue, PeakValue float64
    Duration, Interval time.Duration
}

type ConstantNoisePattern struct { // 常量+噪声
    Base, NoiseAmplitude float64
}
```

**职责**：
- 根据配置创建数据模式实例
- 按时间序列生成数据点
- 支持多数据点组合（一个设备多个属性）

### 3.3 配置系统

#### 全局配置 (simulator.yaml)

```yaml
simulator:
  log_level: info           # 日志级别
  log_format: json          # 日志格式 (json/text)
  
  # MQTT 全局配置
  mqtt:
    broker: localhost:1883
    username: ""
    password: ""
    client_id_prefix: "sim-"
  
  # TCP 全局配置
  tcp:
    host: localhost
    port: 9000
    
  # Modbus 全局配置
  modbus:
    host: localhost
    port: 502
    
  # OPC UA 全局配置
  opcua:
    endpoint: "opc.tcp://localhost:4840"
```

#### 设备模板 (devices/thermometer.yaml)

```yaml
device_template:
  name: "温湿度传感器"
  model_id: "sensor_th01"
  data_points:
    - name: temperature
      unit: "°C"
      pattern:
        type: sine
        amplitude: 5.0
        period: 3600      # 1小时周期
        offset: 25.0       # 基准温度
        noise: 0.5         # 噪声幅度
      range: [-10, 50]
      
    - name: humidity
      unit: "%RH"
      pattern:
        type: random_walk
        min: 30.0
        max: 80.0
        step: 0.5
      range: [0, 100]
      
    - name: battery
      unit: "%"
      pattern:
        type: constant_noise
        base: 85.0
        noise_amplitude: 2.0
      range: [0, 100]
      
  report_interval: 5s      # 上报间隔
  metadata:
    manufacturer: "SimuSensor Inc."
    firmware: "v2.1.0"
```

#### 场景配置 (scenarios/mqtt_sim.yaml)

```yaml
scenario:
  name: "MQTT 温湿度传感器集群"
  protocol: mqtt
  enabled: true
  
  devices:
    - template: thermometer
      count: 5
      id_prefix: "th-sensor"
      domain_id: "factory-01"
      region: "workshop-a"
      
    - template: motor
      count: 3
      id_prefix: "motor"
      domain_id: "factory-01"
      region: "workshop-b"
      
  mqtt:
    topic_format: "devices/{domain_id}/{region}/{device_id}/telemetry"
    qos: 1
    
  runtime:
    duration: 0            # 0 = 无限运行
    start_delay: 0         # 启动延迟
    stagger: 100ms         # 设备启动间隔（避免同时连接）
```

### 3.4 协议适配器

#### MQTT 适配器

```go
type MQTTAdapter struct {
    config   MQTTSimConfig
    client   mqtt.Client  // 使用 paho mqtt 客户端
    logger   *zerolog.Logger
}

func (a *MQTTAdapter) SendData(deviceID string, data []byte) error {
    topic := formatTopic(a.config.TopicFormat, deviceID)
    token := a.client.Publish(topic, a.config.QoS, false, data)
    token.Wait()
    return token.Error()
}
```

#### TCP 客户端适配器

```go
type TCPClientAdapter struct {
    config  TCPSimConfig
    conn    net.Conn
    logger  *zerolog.Logger
}

func (a *TCPClientAdapter) SendData(deviceID string, data []byte) error {
    frame := EncodeFrame(FrameTypeData, data)
    _, err := a.conn.Write(frame)
    return err
}
```

#### TCP 服务端适配器

```go
type TCPServerAdapter struct {
    config    TCPSimConfig
    listener  net.Listener
    clients   map[string]net.Conn  // deviceID -> connection
    logger    *zerolog.Logger
}

func (a *TCPServerAdapter) SendData(deviceID string, data []byte) error {
    conn, ok := a.clients[deviceID]
    if !ok {
        return fmt.Errorf("device %s not connected", deviceID)
    }
    frame := EncodeFrame(FrameTypeData, data)
    _, err := conn.Write(frame)
    return err
}
```

#### Modbus 从站适配器

```go
type ModbusSlaveAdapter struct {
    config    ModbusSimConfig
    listener  net.Listener
    registers map[uint16][]byte  // 寄存器地址 -> 数据
    logger    *zerolog.Logger
}

func (a *ModbusSlaveAdapter) SetRegisters(addr uint16, values []byte) {
    a.registers[addr] = values
}

// 内部处理 Modbus 读取请求，返回寄存器数据
func (a *ModbusSlaveAdapter) handleReadRequest(conn net.Conn, frame MBAPFrame) {
    // 解析功能码和地址，返回对应的寄存器数据
}
```

#### Modbus 主站适配器

```go
type ModbusMasterAdapter struct {
    config  ModbusSimConfig
    conn    net.Conn
    logger  *zerolog.Logger
}

func (a *ModbusMasterAdapter) ReadRegisters(slaveID byte, addr uint16, quantity uint16) ([]byte, error) {
    frame := BuildReadRequest(slaveID, addr, quantity)
    _, err := a.conn.Write(frame)
    // 读取响应...
}
```

#### OPC UA 服务器适配器

```go
type OPCUAServerAdapter struct {
    config   OPCUASimConfig
    server   *opcua.Server
    nodes    map[string]*SimNode  // nodeID -> 模拟节点
    logger   *zerolog.Logger
}

type SimNode struct {
    NodeID    string
    Value     float64
    Generator Pattern
    mu        sync.RWMutex
}

func (a *OPCUAServerAdapter) Start(ctx context.Context) error {
    // 启动 OPC UA 服务器
    // 注册节点
    // 启动数据更新协程
}

func (a *OPCUAServerAdapter) updateNodes() {
    for _, node := range a.nodes {
        node.mu.Lock()
        node.Value = node.Generator.Next(time.Now())
        node.mu.Unlock()
        // 更新 OPC UA 服务器中的节点值
    }
}
```

---

## 4. 启动方式

### 4.1 命令行启动

```bash
# 启动 MQTT 模拟器
go run cmd/simulator/main.go --protocol mqtt --config configs/simulator/scenarios/mqtt_sim.yaml

# 启动 TCP 模拟器（客户端模式）
go run cmd/simulator/main.go --protocol tcp --mode client --config configs/simulator/scenarios/tcp_sim.yaml

# 启动 TCP 模拟器（服务端模式）
go run cmd/simulator/main.go --protocol tcp --mode server --config configs/simulator/scenarios/tcp_sim.yaml

# 启动 Modbus 模拟器（从站模式）
go run cmd/simulator/main.go --protocol modbus --mode slave --config configs/simulator/scenarios/modbus_sim.yaml

# 启动 Modbus 模拟器（主站模式）
go run cmd/simulator/main.go --protocol modbus --mode master --config configs/simulator/scenarios/modbus_sim.yaml

# 启动 OPC UA 模拟器
go run cmd/simulator/main.go --protocol opcua --config configs/simulator/scenarios/opcua_sim.yaml
```

### 4.2 CLI 参数

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--protocol` | `-p` | 协议类型 (mqtt/tcp/modbus/opcua) | 必填 |
| `--config` | `-c` | 场景配置文件路径 | 必填 |
| `--mode` | `-m` | 模式 (client/server, master/slave) | 协议默认 |
| `--devices` | `-d` | 设备数量（覆盖配置） | 配置值 |
| `--interval` | `-i` | 发送间隔（覆盖配置） | 配置值 |
| `--duration` | `-D` | 运行时长（0=无限） | 0 |
| `--verbose` | `-v` | 详细日志 | false |

### 4.3 Makefile 快捷命令

```makefile
# 模拟器相关
sim-mqtt:               ## 启动 MQTT 模拟器
	go run cmd/simulator/main.go -p mqtt -c configs/simulator/scenarios/mqtt_sim.yaml

sim-tcp-client:         ## 启动 TCP 客户端模拟器
	go run cmd/simulator/main.go -p tcp -m client -c configs/simulator/scenarios/tcp_sim.yaml

sim-tcp-server:         ## 启动 TCP 服务端模拟器
	go run cmd/simulator/main.go -p tcp -m server -c configs/simulator/scenarios/tcp_sim.yaml

sim-modbus-slave:       ## 启动 Modbus 从站模拟器
	go run cmd/simulator/main.go -p modbus -m slave -c configs/simulator/scenarios/modbus_sim.yaml

sim-modbus-master:      ## 启动 Modbus 主站模拟器
	go run cmd/simulator/main.go -p modbus -m master -c configs/simulator/scenarios/modbus_sim.yaml

sim-opcua:              ## 启动 OPC UA 模拟器
	go run cmd/simulator/main.go -p opcua -c configs/simulator/scenarios/opcua_sim.yaml

sim-all:                ## 启动所有模拟器（后台）
	@echo "Starting all simulators..."
	@make sim-mqtt &
	@make sim-tcp-client &
	@make sim-modbus-slave &
	@make sim-opcua &
	@echo "All simulators started. Press Ctrl+C to stop."
	@wait

sim-stop:               ## 停止所有模拟器
	pkill -f "cmd/simulator" || true
```

---

## 5. 日志设计

### 5.1 日志格式

使用 `zerolog` 输出结构化 JSON 日志：

```json
{
  "level": "info",
  "time": "2026-08-20T10:30:00Z",
  "component": "simulator",
  "protocol": "mqtt",
  "device_id": "th-sensor-001",
  "event": "data_sent",
  "topic": "devices/factory-01/workshop-a/th-sensor-001/telemetry",
  "bytes": 128
}
```

### 5.2 日志事件类型

| 事件 | 级别 | 说明 |
|------|------|------|
| `simulator_start` | info | 模拟器启动 |
| `simulator_stop` | info | 模拟器停止 |
| `device_created` | info | 设备实例创建 |
| `device_connected` | info | 设备连接成功 |
| `device_disconnected` | error | 设备断开（触发退出） |
| `data_sent` | debug | 数据发送成功 |
| `data_send_error` | error | 数据发送失败（触发退出） |
| `config_loaded` | info | 配置加载完成 |
| `config_error` | error | 配置解析错误 |

---

## 6. 测试策略

### 6.1 单元测试

| 组件 | 测试内容 |
|------|----------|
| `generator.go` | 各种模式的数据生成正确性、边界值 |
| `devices/patterns.go` | 正弦波周期、随机游走范围、阶梯变化时序 |
| `mqtt/codec.go` | MQTT 报文编解码 |
| `tcp/frame.go` | TCP 帧编解码、边界情况 |
| `modbus/frame.go` | Modbus MBAP 帧编解码 |
| `config.go` | 配置文件加载、默认值、验证 |

### 6.2 集成测试

| 场景 | 测试内容 |
|------|----------|
| MQTT 端到端 | 模拟器 → Broker → MQTT 网关 → NATS |
| TCP 客户端端到端 | 模拟器 → TCP 网关 → NATS |
| TCP 服务端端到端 | TCP 网关 → 模拟器 |
| Modbus 从站端到端 | Modbus 网关 → 模拟器 → NATS |
| Modbus 主站端到端 | 模拟器 → Modbus 网关 |
| OPC UA 端到端 | 模拟器(OPC UA Server) → OPC UA 网关 → NATS |

### 6.3 测试文件结构

```
internal/simulator/
├── generator_test.go
├── config_test.go
├── devices/
│   └── patterns_test.go
├── mqtt/
│   └── codec_test.go
├── tcp/
│   └── frame_test.go
└── modbus/
    └── frame_test.go
```

---

## 7. 依赖项

### 7.1 新增依赖

| 依赖 | 用途 |
|------|------|
| `github.com/rs/zerolog` | 结构化日志 |
| `github.com/eclipse/paho.mqtt.golang` | MQTT 客户端 |
| `github.com/gopcua/opcua` | OPC UA 服务器（已有客户端依赖） |
| `gopkg.in/yaml.v3` | YAML 配置解析 |

### 7.2 复用现有代码

| 包 | 复用内容 |
|---|----------|
| `internal/message` | DeviceEnvelope、MessageUnit 结构体 |
| `internal/gateway` | GatewayAdapter、Publisher 接口 |
| `pkg/nats` | NATS 主题构建、JetStream 发布 |
| `internal/modbus/frame.go` | Modbus MBAP 帧格式定义 |
| `internal/tcp/frame.go` | TCP 帧格式定义 |

---

## 8. 实现计划

### 阶段一：基础框架（2天）

- [ ] 创建 `internal/simulator/` 目录结构
- [ ] 实现配置加载系统 (`config.go`)
- [ ] 实现数据生成器引擎 (`generator.go`, `devices/patterns.go`)
- [ ] 实现模拟引擎核心 (`engine.go`)
- [ ] 创建 `cmd/simulator/main.go` 入口

### 阶段二：MQTT 模拟器（1天）

- [ ] 实现 MQTT 客户端模拟器
- [ ] 实现 MQTT 报文编解码
- [ ] 创建 MQTT 示例配置和设备模板
- [ ] 编写单元测试

### 阶段三：TCP 模拟器（1.5天）

- [ ] 实现 TCP 客户端模拟器
- [ ] 实现 TCP 服务端模拟器
- [ ] 实现 TCP 帧编解码（复用现有格式）
- [ ] 创建 TCP 示例配置
- [ ] 编写单元测试

### 阶段四：Modbus 模拟器（2天）

- [ ] 实现 Modbus 从站模拟器
- [ ] 实现 Modbus 主站模拟器
- [ ] 实现 Modbus MBAP 帧处理
- [ ] 创建 Modbus 示例配置
- [ ] 编写单元测试

### 阶段五：OPC UA 模拟器（2天）

- [ ] 实现内嵌 OPC UA 服务器
- [ ] 实现节点定义与数据绑定
- [ ] 创建 OPC UA 示例配置
- [ ] 编写单元测试

### 阶段六：集成与文档（1天）

- [ ] 更新 Makefile 添加模拟器命令
- [ ] 创建完整示例配置集
- [ ] 编写集成测试
- [ ] 更新项目文档

**总工期：约 9.5 天**

---

## 9. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| OPC UA 服务器库复杂 | OPC UA 模拟器实现困难 | 使用 gopcua 的 server 示例作为基础，简化功能 |
| 并发连接数限制 | 大量设备模拟时性能问题 | 使用连接池、批量发送、可配置并发数 |
| 配置格式频繁变更 | 用户使用成本高 | 提供配置验证工具和示例模板 |
| 依赖库版本兼容 | 构建失败 | 锁定依赖版本，定期更新测试 |

---

## 10. 附录

### 10.1 配置示例：完整温湿度传感器

```yaml
# configs/simulator/devices/thermometer.yaml
device_template:
  name: "温湿度传感器"
  model_id: "sensor_th01"
  
  data_points:
    - name: temperature
      unit: "°C"
      description: "环境温度"
      pattern:
        type: sine
        amplitude: 5.0
        period: 3600
        offset: 25.0
        phase: 0
        noise: 0.5
      range: [-10, 50]
      precision: 1
      
    - name: humidity
      unit: "%RH"
      description: "相对湿度"
      pattern:
        type: random_walk
        min: 30.0
        max: 80.0
        step: 0.5
      range: [0, 100]
      precision: 1
      
    - name: battery
      unit: "%"
      description: "电池电量"
      pattern:
        type: constant_noise
        base: 85.0
        noise_amplitude: 2.0
      range: [0, 100]
      precision: 0
      
  report_interval: 5s
  
  metadata:
    manufacturer: "SimuSensor Inc."
    firmware: "v2.1.0"
    protocol_version: "1.0"
```

### 10.2 配置示例：MQTT 场景

```yaml
# configs/simulator/scenarios/mqtt_sim.yaml
scenario:
  name: "MQTT 传感器集群模拟"
  protocol: mqtt
  enabled: true
  
  devices:
    - template: thermometer
      count: 10
      id_prefix: "th-sensor"
      domain_id: "factory-01"
      region: "workshop-a"
      
    - template: motor
      count: 5
      id_prefix: "motor"
      domain_id: "factory-01"
      region: "workshop-b"
      
  mqtt:
    broker: "localhost:1883"
    username: ""
    password: ""
    client_id_prefix: "sim-"
    topic_format: "devices/{domain_id}/{region}/{device_id}/telemetry"
    qos: 1
    clean_session: true
    
  runtime:
    duration: 0
    start_delay: 0
    stagger: 100ms
    
  logging:
    level: info
    format: json
```

---

**文档完成** | 共 24 项设计决策已确认
