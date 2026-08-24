# IoT 数据中台

多协议设备接入、实时数据处理、时序存储与可视化管理的一体化平台。

## 功能特性

- **多协议网关** — MQTT 3.1.1 / TCP 自定义帧 / Modbus TCP / OPC UA 四种协议同时接入
- **设备管理** — 设备注册、认证、在线/离线状态跟踪
- **物模型** — 属性/命令/事件三层结构定义设备数据模型
- **规则引擎** — Pipeline 模式数据处理：过滤、转换、条件、聚合、JavaScript 脚本
- **时序存储** — TDengine 存储设备上报数据，支持按时间/设备/聚合查询
- **告警通知** — Webhook 通知 + 5 分钟去重 + 每分钟限流 + 3 次退避重试
- **业务域隔离** — 多租户隔离，域级别设备/模型/规则/告警独立
- **管理控制台** — Vue 3 + Element Plus Web 界面，6 个功能模块，全实体 CRUD
- **规则可视化编辑器** — Vue Flow 画布模式，拖拽节点 + 连线 + 右侧配置面板
- **测试数据模拟器** — 多协议模拟器，YAML 配置驱动，无需真实设备即可测试

## 架构

```
Device → Gateway (MQTT/TCP/Modbus) → NATS JetStream → Rule Engine → Timeseries Writer (TDengine)
                                                          ↓
                                                    Alert Service (Webhooks)
```

### 微服务分解（9 个服务）

| 服务 | 端口 | 启动方式 | 职责 |
|------|------|---------|------|
| console | 8080 | `make start` | HTTP API + 管理控制台后端 |
| device-service | 8081 | `make start` | 设备管理 + Redis 状态 + 认证 |
| alert-service | 8082 | `make start` | 告警通知 + NATS 订阅 + Webhook |
| rule-engine | — | `make start` | 规则引擎 + NATS 消费 |
| timeseries-writer | — | `make start` | TDengine 时序写入 + NATS 消费 |
| mqtt-gateway | 1883 | 网页启动 | MQTT 协议网关 |
| tcp-gateway | 9000 | 网页启动 | TCP 自定义协议网关 |
| modbus-gateway | 502 | 网页启动 | Modbus TCP 工业协议网关 |
| opcua-gateway | 4840 | 网页启动 | OPC UA 工业协议网关 |

> 后端 5 个服务通过 `make start` 统一启动；4 个协议网关通过管理控制台网页启停。

## 技术栈

| 层 | 技术 |
|------|------|
| 后端 | Go 1.22+、Gin、GORM |
| 前端 | Vue 3（Composition API）、Element Plus、Vite |
| 消息 | NATS JetStream |
| 时序库 | TDengine（REST API） |
| 缓存 | Redis 7 |
| 数据库 | SQLite（开发）/ MySQL / PostgreSQL（生产） |
| 脚本引擎 | Goja（JavaScript VM） |

## 快速开始

### 前置条件

- Go ≥ 1.22
- Node.js ≥ 18
- Docker & Docker Compose

### 一键启动

```bash
make build    # 编译所有服务
make start    # 启动基础设施 + 5 个后端服务
```

> 首次启动 Console 会自动创建 SQLite 数据库、建表并填充种子数据（2 个域、5 个物模型、14 个设备、12 个绑定、4 条规则、3 个 Webhook、4 个网关）。

然后启动前端：

```bash
cd web && npm install && npm run dev
```

打开浏览器 **http://localhost:3000**，进入 **网关管理** 页面点击"启动"按钮启用所需协议网关。

### 服务管理

```bash
make start      # 启动所有后端服务（含基础设施）
make stop       # 停止所有服务（含基础设施）
make restart    # 重启所有服务
make status     # 查看服务运行状态
make logs       # 查看所有服务日志
make log-console          # 查看指定服务日志
make log-alert-service    # 查看告警服务日志
```

### 单独启动（调试用）

```bash
go run ./cmd/console              # 仅启动控制台 :8080
go run ./cmd/device-service       # 仅启动设备服务 :8081
go run ./cmd/alert-service        # 仅启动告警服务 :8082
go run ./cmd/rule-engine          # 仅启动规则引擎
go run ./cmd/timeseries-writer    # 仅启动时序写入
```

> 网关由 Console 通过网页管理，不需要单独启动。

## 项目结构

```
dataCenter/
├── cmd/                        # 微服务入口 + 模拟器
│   ├── console/                # HTTP API + 管理控制台（:8080）
│   ├── mqtt-gateway/           # MQTT 网关（:1883）
│   ├── tcp-gateway/            # TCP 网关（:9000）
│   ├── modbus-gateway/         # Modbus 网关（:502）
│   ├── device-service/         # 设备管理 + Redis 状态
│   ├── rule-engine/            # 规则引擎 + NATS 消费
│   ├── timeseries-writer/      # 时序写入 + NATS 消费
│   ├── alert-service/          # 告警服务
│   └── simulator/              # 多协议测试数据模拟器
├── api/v1/                     # REST API Handler 层
│   ├── device_handler.go       # 设备管理
│   ├── domain_handler.go       # 业务域管理
│   ├── model_handler.go        # 物模型管理
│   ├── rule_handler.go         # 规则引擎
│   └── alert_handler.go        # 告警管理
├── internal/                   # 核心业务逻辑
│   ├── device/                 # 设备管理（GORM + Redis 状态）
│   ├── domain/                 # 业务域 + 成员 RBAC
│   ├── model/                  # 物模型 + 设备绑定
│   ├── rule/                   # 规则引擎 + 持久化
│   ├── alert/                  # 告警 + Webhook 发送
│   ├── timeseries/             # TDengine 时序读写
│   ├── mqtt/                   # MQTT 3.1.1 协议实现
│   ├── tcp/                    # TCP 自定义帧协议
│   ├── modbus/                 # Modbus TCP 协议
│   ├── gateway/                # 网关抽象层
│   ├── message/                # 统一消息格式
│   └── simulator/              # 模拟器引擎 + 协议适配器
│       ├── engine.go           # 模拟引擎核心
│       ├── config.go           # 配置加载
│       ├── devices/            # 设备模板 + 数据模式
│       ├── mqtt/               # MQTT 模拟适配器
│       ├── tcp/                # TCP 模拟适配器
│       ├── modbus/             # Modbus 模拟适配器
│       └── opcua/              # OPC UA 模拟适配器
├── pkg/nats/                   # NATS 客户端封装
├── proto/                      # Protobuf 定义
├── web/                        # Vue 3 前端
│   └── src/
│       ├── views/              # 6 个页面
│       ├── api/                # API 客户端
│       ├── stores/             # Pinia 状态
│       └── router/             # 路由配置
├── configs/                    # 配置文件
│   └── simulator/              # 模拟器配置
│       ├── simulator.yaml      # 全局配置
│       ├── devices/            # 设备模板（4 种）
│       └── scenarios/          # 场景配置（4 种协议）
├── deploy/                     # Docker Compose
├── docs/                       # 文档
│   ├── api.md                  # API 文档（32 个端点）
│   ├── deployment.md           # 部署文档
│   └── user-guide.md           # 用户手册
├── tests/                      # 集成测试
└── Makefile
```

## 文档

| 文档 | 说明 |
|------|------|
| [API 文档](docs/api.md) | 32 个 REST API 端点完整说明 |
| [部署文档](docs/deployment.md) | 开发环境、手动部署、生产配置、故障排查 |
| [用户手册](docs/user-guide.md) | 管理控制台 6 个模块操作指南 |
| [模拟器设计](docs/design-test-simulator.md) | 测试数据模拟器架构与配置说明 |
| [设计文档](docs/superpowers/specs/2026-08-19-data-center-design.md) | 整体架构设计 |
| [实施计划](docs/superpowers/specs/implementation-plan.md) | 8 阶段实施路线 |

## API 概览

| 模块 | 端点 | 说明 |
|------|------|------|
| 设备 | `POST/GET/PUT/DELETE /api/v1/devices` | CRUD + Token 验证 |
| 域管理 | `POST/GET/DELETE /api/v1/domains` | 域 CRUD + 成员管理 |
| 物模型 | `POST/GET/DELETE /api/v1/models` | 模型 CRUD + 设备绑定 |
| 规则引擎 | `POST/GET/PUT/DELETE /api/v1/rules` | 规则 CRUD + 启用/禁用 |
| 告警 | `POST/GET/PUT/DELETE /api/v1/alerts/webhooks` | Webhook 管理 + 测试 + 日志 |

完整 API 文档见 [docs/api.md](docs/api.md)。

## Docker Compose 基础设施

```yaml
services:
  nats:       # 消息总线（JetStream 持久化）    → :4222
  tdengine:   # 时序数据库（REST API）          → :6041
  redis:      # 缓存 + 设备在线状态             → :6379
```

基础设施随 `make start` 自动启动，也可单独管理：

```bash
make docker-up      # 启动基础设施
make docker-down    # 停止基础设施
make docker-logs    # 查看容器日志
```

## 测试

```bash
go test ./... -v -count=1            # 运行所有测试
go test ./internal/message/ -v       # 消息格式测试
go test ./pkg/nats/ -v               # NATS Topic 测试
go test ./tests/ -v                  # 集成测试
```

## 模拟测试

项目内置多协议测试数据模拟器，无需真实设备即可模拟完整的设备数据流。

### 启动模拟测试（完整流程）

```bash
# 1. 编译 + 启动所有后端服务
make build && make start

# 2. 启动前端（新终端）
cd web && npm run dev

# 3. 浏览器 http://localhost:3000 → 网关管理 → 启动对应网关
#    - MQTT 模拟 → 启动 "MQTT 主网关"（端口 1883）
#    - TCP 模拟  → 启动 "TCP 工业网关"（端口 9000）
#    - Modbus 模拟 → 启动 "Modbus 网关"（端口 502）
#    - OPC UA 模拟 → 启动 "OPC UA 网关"（端口 4840）

# 4. 启动模拟器（新终端，选择对应协议）
make sim-mqtt

# 5. 回到前端查看模拟数据（设备列表、监控面板等）
```

> **注意：** 模拟器必须在对应网关启动后才能连接。如果模拟器报连接失败，请确认网关已在前端页面中启动。

### 可用模拟器

| 命令 | 协议 | 场景 |
|------|------|------|
| `make sim-mqtt` | MQTT | 10 个传感器集群（温湿度、电机、电表） |
| `make sim-tcp-client` | TCP | TCP 客户端模式模拟 |
| `make sim-tcp-server` | TCP | TCP 服务端模式模拟 |
| `make sim-modbus-slave` | Modbus | Modbus 从站模拟 |
| `make sim-modbus-master` | Modbus | Modbus 主站模拟 |
| `make sim-opcua` | OPC UA | OPC UA 服务器模拟 |
| `make sim-stop` | — | 停止所有模拟器 |

### 直接运行（自定义参数）

```bash
# 指定协议和配置文件
go run cmd/simulator/main.go --protocol mqtt --config configs/simulator/scenarios/mqtt_sim.yaml

# 自定义设备数量和发送间隔
go run cmd/simulator/main.go --protocol mqtt --config configs/simulator/scenarios/mqtt_sim.yaml --devices 20 --interval 3s

# 运行指定时长（秒）
go run cmd/simulator/main.go --protocol mqtt --config configs/simulator/scenarios/mqtt_sim.yaml --duration 60

# 详细日志
go run cmd/simulator/main.go --protocol mqtt --config configs/simulator/scenarios/mqtt_sim.yaml --verbose
```

**CLI 参数：**

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--protocol` | 协议类型 (mqtt/tcp/modbus/opcua) | 必填 |
| `--config` | 场景配置文件路径 | 必填 |
| `--mode` | 模式 (client/server, master/slave) | 协议默认 |
| `--devices` | 设备数量（覆盖配置） | 配置值 |
| `--interval` | 发送间隔（覆盖配置） | 配置值 |
| `--duration` | 运行时长，0=无限（秒） | 0 |
| `--verbose` | 详细日志 | false |

### 设备模板

模拟器内置 4 种设备模板，定义在 `configs/simulator/devices/` 下：

| 模板 | 文件 | 数据点 | 数据模式 |
|------|------|--------|----------|
| 温湿度传感器 | `thermometer.yaml` | 温度、湿度、电池 | 正弦波 + 随机游走 |
| 电机 | `motor.yaml` | 转速、振动、温度 | 正弦波 + 常量噪声 |
| 电表 | `power_meter.yaml` | 电压、电流、功率 | 阶梯变化 |
| GPS 追踪器 | `gps_tracker.yaml` | 经度、纬度、速度 | 随机游走 |

### 场景配置

场景配置定义在 `configs/simulator/scenarios/` 下，每个文件描述一个模拟场景：

```yaml
# configs/simulator/scenarios/mqtt_sim.yaml 示例
scenario:
  name: "MQTT 传感器集群模拟"
  protocol: mqtt
  devices:
    - template: thermometer    # 使用 thermometer 模板
      count: 5                 # 创建 5 个实例
      id_prefix: "th-sensor"   # 设备 ID 前缀
      domain_id: "factory-01"
      region: "workshop-a"
    - template: motor
      count: 3
      id_prefix: "motor"
  mqtt:
    broker: "localhost:1883"
    topic_format: "devices/{domain_id}/{region}/{device_id}/telemetry"
  runtime:
    duration: 0                # 0 = 无限运行
    stagger: 100ms             # 设备启动间隔
```

### 数据流验证

模拟器启动后，数据按以下路径流转：

```
模拟器 → 协议网关 → NATS JetStream → 规则引擎 → 时序写入（TDengine）
                                          ↓
                                    告警服务（Webhooks）
```

在管理控制台中可以验证：
1. **设备列表** — 查看模拟设备是否自动注册
2. **规则引擎** — 创建规则，观察节点链处理效果
3. **监控面板** — 查看网关连接数和数据吞吐量

## 开发计划

| 阶段 | 状态 | 说明 |
|------|------|------|
| 阶段一：基础设施 + 消息层 | ✅ 完成 | Docker Compose、Protobuf、NATS 封装 |
| 阶段二：协议网关 | ✅ 完成 | MQTT / TCP / Modbus 三协议实现 |
| 阶段三：设备管理 | ✅ 完成 | 认证、状态、REST API |
| 阶段四：规则引擎 | ✅ 完成 | Pipeline + 节点 + Goja 脚本 |
| 阶段五：物模型 + 时序 | ✅ 完成 | 模型定义 + TDengine 读写 |
| 阶段六：告警 + 域管理 | ✅ 完成 | Webhook + 多租户隔离 |
| 阶段七：管理控制台 | ✅ 完成 | Vue 3 前端 6 个功能页面 |
| 阶段八：集成测试 + 文档 | ✅ 完成 | 测试 + API/部署/用户文档 |

### 待完善

- 生产环境数据库迁移（MySQL / PostgreSQL）
- 可观测性体系（Metrics / Tracing / Logging）

## License

MIT
