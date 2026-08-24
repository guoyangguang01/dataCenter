# IoT 数据中台 — 架构文档

## 1. 系统总览

IoT 数据中台是一个基于 Go 微服务的多协议设备接入平台，支持 MQTT、TCP、Modbus、OPC-UA 四种工业协议，通过 NATS JetStream 实现消息分发，规则引擎实时处理数据，TDengine 存储时序数据，Webhook 告警推送。

```
┌─────────────────────────────────────────────────────────────────┐
│                        Vue 3 管理控制台                          │
│  Dashboard │ Devices │ Rules │ Models │ Alerts │ Gateways       │
└────────────────────────────┬────────────────────────────────────┘
                             │ HTTP REST API (:8080)
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Console API Server                           │
│  Gin Router + GORM(SQLite) + 全部业务服务 + 网关管理              │
└──────┬──────────────┬──────────────┬──────────────┬─────────────┘
       │              │              │              │
       ▼              ▼              ▼              ▼
  ┌─────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐
  │  NATS   │  │ TDengine │  │  Redis   │  │   SQLite     │
  │JetStream│  │ 时序存储  │  │ 状态缓存  │  │  业务数据     │
  └─────────┘  └──────────┘  └──────────┘  └──────────────┘
```

---

## 2. 微服务拆分 (8 个二进制)

| 服务 | 入口 | 端口 | 状态 | 职责 |
|---|---|---|---|---|
| **console** | `cmd/console/main.go` | :8080 | ✅ 完整 | HTTP API 服务器，集成全部业务逻辑 |
| **mqtt-gateway** | `cmd/mqtt-gateway/main.go` | :1883 | ✅ 完整 | MQTT 3.1.1 协议网关 |
| **tcp-gateway** | `cmd/tcp-gateway/main.go` | :9000 | ✅ 完整 | TCP 二进制帧协议网关 |
| **modbus-gateway** | `cmd/modbus-gateway/main.go` | :502 | ✅ 完整 | Modbus TCP 工业协议网关 |
| **timeseries-writer** | `cmd/timeseries-writer/main.go` | — | ✅ 完整 | NATS 消费者 → TDengine 批量写入 |
| **rule-engine** | `cmd/rule-engine/main.go` | — | ✅ 完整 | NATS 消费者 → 规则链执行 |
| **alert-service** | `cmd/alert-service/main.go` | — | ⬚ 桩 | 告警服务（未接线） |
| **device-service** | `cmd/device-service/main.go` | — | ⬚ 桩 | 设备服务（未接线） |

---

## 3. 核心数据流

```
┌──────────┐
│ 物理设备  │
└────┬─────┘
     │ MQTT / TCP / Modbus / OPC-UA
     ▼
┌─────────────────┐
│   协议网关       │  解析协议帧 → 构建 DeviceEnvelope
│  GatewayAdapter  │  更新设备在线状态
└────┬────────────┘
     │ PublishEnvelope()
     ▼
┌─────────────────────────────────────────────────────────┐
│              NATS JetStream (DEVICE_DATA)                │
│         主题: domains.{域}.devices.{地区}.{类型}.{ID}.up   │
│         保留: 7天                                         │
└──────┬──────────────────┬──────────────────┬────────────┘
       │                  │                  │
       ▼                  ▼                  ▼
┌──────────────┐  ┌───────────────┐  ┌──────────────┐
│  规则引擎     │  │ 时序写入器     │  │  控制台       │
│  Pipeline    │  │ 批量写入       │  │  实时展示     │
│  Filter→     │  │ TDengine      │  │  (未来)       │
│  Transform→  │  │ 每设备子表     │  │              │
│  Condition→  │  └───────────────┘  └──────────────┘
│  Action      │
└──────┬───────┘
       │ 告警事件
       ▼
┌──────────────┐
│  Webhook     │  HTTP POST (钉钉/飞书/自定义)
│  去重+限流    │  5分钟去重窗口，10条/分钟限流
└──────────────┘
```

---

## 4. 包依赖关系图

```
                          ┌─────────────┐
                          │  pkg/nats   │
                          │ JetStream   │
                          │ Topic构建    │
                          └──────┬──────┘
                                 │
              ┌──────────────────┼──────────────────┐
              │                  │                  │
              ▼                  ▼                  ▼
    ┌─────────────────┐ ┌──────────────┐ ┌──────────────────┐
    │ internal/gateway │ │internal/message│ │ internal/rule    │
    │ GatewayAdapter   │ │ DeviceEnvelope │ │ Engine/Pipeline  │
    │ Publisher        │ │ MessageUnit    │ │ Node/Registry    │
    │ Launcher         │ └──────┬───────┘ │ ScriptNode(goja) │
    │ NATSPublisher    │        │         └────────┬─────────┘
    └───┬──┬──┬──┬────┘        │                  │
        │  │  │  │              │                  │
        │  │  │  └──────────────┼────┬─────────────┘
        │  │  │                 │    │
        ▼  ▼  ▼                 ▼    ▼
┌──────┐┌──────┐┌──────┐  ┌─────────────────┐
│ mqtt ││ tcp  ││modbus│  │  internal/alert  │
│协议栈 ││帧协议 ││MBAP  │  │  WebhookSender  │
│Codec ││      ││      │  │  去重/限流        │
└──────┘└──────┘└──────┘  └─────────────────┘

    ┌───────────────┐     ┌──────────────┐
    │internal/device│     │internal/domain│
    │ CRUD/认证/状态 │     │ 域/成员管理    │
    └───────┬───────┘     └──────────────┘
            │
            ▼
    ┌──────────────┐     ┌──────────────────┐
    │internal/model│     │internal/timeseries│
    │ 物模型/绑定   │     │ Writer (TDengine) │
    └──────────────┘     │ QueryService      │
                         └──────────────────┘
```

---

## 5. 核心接口关系

### 5.1 GatewayAdapter（协议网关抽象）

```go
type GatewayAdapter interface {
    Start() error                                    // 启动监听
    Stop() error                                     // 停止监听
    OnDeviceStatusChanged(deviceID, status)          // 设备状态变更回调
}
```

```
                    GatewayAdapter
                         ▲
          ┌──────────────┼──────────────┐
          │              │              │
    ┌─────┴─────┐  ┌─────┴─────┐  ┌────┴──────┐
    │mqtt.Gateway│  │tcp.Gateway│  │modbus.Gateway│
    │ :1883      │  │ :9000     │  │ :502        │
    │ MQTT 3.1.1 │  │ 自定义帧   │  │ MBAP头       │
    └────────────┘  └───────────┘  └─────────────┘
```

### 5.2 Publisher（消息发布抽象）

```go
type Publisher interface {
    PublishEnvelope(env *message.DeviceEnvelope) error
}
```

```
                     Publisher
                        ▲
           ┌────────────┼────────────┐
           │            │            │
    ┌──────┴──────┐ ┌───┴────┐ ┌────┴─────┐
    │NATSPublisher│ │Simple  │ │Log       │
    │ JetStream   │ │NATS    │ │Publisher │
    │ (生产模式)   │ │(控制台) │ │(调试回退) │
    └─────────────┘ └────────┘ └──────────┘
```

### 5.3 Node（规则节点抽象）

```go
type Node interface {
    ID() string
    Type() string
    Execute(ctx, env, state) (*DeviceEnvelope, error)
}
```

```
         Pipeline 执行流
         ┌─────────┐
    ────▶│FilterNode│──── 匹配 topic (eq/contains/prefix)
         └────┬────┘
              │
         ┌────▼─────────┐
         │TransformNode │──── 提取指定 topic 的 payload
         └────┬─────────┘
              │
         ┌────▼─────────┐
         │ConditionNode │──── 条件判断
         └────┬─────────┘
              │
         ┌────▼─────────┐
         │AggregateNode │──── 滑动窗口聚合
         └────┬─────────┘
              │
    ┌─────────▼──────────┐
    │    ActionNode      │──── 发布到 NATS / Webhook
    │  或 ScriptNode     │──── JavaScript 脚本 (goja)
    └────────────────────┘
```

---

## 6. 前端架构

### 6.1 页面 → API → 后端服务 映射

```
┌─────────────────────────────────────────────────────────────────┐
│                        Vue 3 前端                                │
│                                                                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐            │
│  │Dashboard │ │ Devices  │ │  Rules   │ │  Models  │            │
│  │ 统计概览  │ │ 设备管理  │ │ 规则配置  │ │ 物模型    │            │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘            │
│       │            │            │            │                    │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐            │
│  │  Alerts  │ │ Domains  │ │ Gateways │ │DevData   │            │
│  │ 告警管理  │ │ 域管理    │ │ 网关管理  │ │ 实时数据  │            │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘            │
└───────┼────────────┼────────────┼────────────┼───────────────────┘
        │            │            │            │
        ▼            ▼            ▼            ▼
   ┌─────────────────────────────────────────────┐
   │          API Layer (axios)                   │
   │   baseURL: /api/v1                           │
   │   deviceApi / domainApi / modelApi / ...     │
   └──────────────────┬──────────────────────────┘
                      │ HTTP
                      ▼
   ┌──────────────────────────────────────────────┐
   │         Gin Router (/api/v1)                 │
   │                                              │
   │  /devices/*    → DeviceHandler               │
   │  /domains/*    → DomainHandler               │
   │  /models/*     → ModelHandler                │
   │  /rules/*      → RuleHandler                 │
   │  /alerts/*     → AlertHandler                │
   │  /gateways/*   → GatewayHandler              │
   │  /stats/*      → StatsHandler                │
   │  /data/*       → DataHandler                 │
   └──────────────────────────────────────────────┘
```

### 6.2 域筛选机制

所有列表页面统一使用域筛选下拉：

```html
<el-select v-model="domainFilter" placeholder="选择域" clearable @change="loadXxx">
  <el-option v-for="d in domains" :key="d.id" :label="d.name" :value="d.id" />
</el-select>
```

- 选择域 → API 请求带 `?domain_id=xxx` → 后端 Where 条件过滤
- 清空选择 → 不传 `domain_id` → 返回全部数据

### 6.3 可视化规则编辑器

```
web/src/components/flow/
├── RuleCanvas.vue      ← 主画布
├── NodePalette.vue     ← 节点拖拽面板
├── FilterNode.vue      ← 过滤节点组件
├── TransformNode.vue   ← 转换节点组件
├── ConditionNode.vue   ← 条件节点组件
├── AggregateNode.vue   ← 聚合节点组件
├── ActionNode.vue      ← 动作节点组件
├── ScriptNode.vue      ← 脚本节点组件
└── config/             ← 各节点配置面板
```

---

## 7. 基础设施依赖

| 组件 | 端口 | 使用者 | 用途 |
|---|---|---|---|
| **SQLite** | 文件 | Console, 各独立服务 | 设备/域/模型/规则/网关持久化 |
| **NATS** | :4222 | Console(发布), 网关(发布), Rule-Engine(消费), Timeseries-Writer(消费) | 消息总线 |
| **NATS JetStream** | 内嵌 | 同上 | 持久化消息流 (DEVICE_DATA 7天, DEVICE_COMMAND 3天, SYSTEM_EVENTS 30天) |
| **TDengine** | :6030 | Timeseries-Writer(写), Console DataHandler(读) | 时序存储 (iot_data 库, 90天保留) |
| **Redis** | :6379 | device.AuthManager, device.StatusManager | 设备认证缓存, 在线状态跟踪 |

### 降级策略

```
NATS 不可用  → Console 回退到 LogPublisher（打印到 stdout）
TDengine 不可用 → DataHandler 返回空数组，Writer 日志告警继续运行
Redis 不可用   → AuthManager/StatusManager 操作静默失败
```

---

## 8. NATS 主题规范

```
六级主题层次:
domains.{domain_id}.devices.{region}.{device_type}.{device_id}.{direction}

方向:
  up   → 设备上报数据
  down → 下发指令

通配符:
  *   → 匹配单级
  >   → 匹配多级

示例:
  domains.factory-a.devices.east.sensor.temp-001.up
  domains.*.devices.*.*.*.up          → 所有设备上报
  domains.factory-a.devices.>         → factory-a 域所有消息
```

### JetStream 流定义

| 流名 | 主题 | 保留策略 | 最大保留 |
|---|---|---|---|
| `DEVICE_DATA` | `domains.*.devices.*.*.*.up` | limits | 7天 |
| `DEVICE_COMMAND` | `domains.*.devices.*.*.*.down` | workqueue | 3天 |
| `SYSTEM_EVENTS` | `system.*` | limits | 30天 |

---

## 9. 设备认证机制

```
两阶段认证:

阶段1: 设备 Token 认证
  设备创建时生成 32字节随机 hex Token
  请求时携带: Authorization: Bearer {token}
  验证: device.Service.VerifyToken(deviceID, token)

阶段2: Domain Key 认证
  格式: {domain_key}:{domain_id}
  用于设备自发现模式
  验证: device.Service.VerifyDomainKey()

ACL 缓存 (Redis):
  缓存键: acl:{deviceID}:{topic}:{action}
  TTL: 5分钟
  策略: 设备只能访问包含自身 ID 的 topic
```

---

## 10. 部署架构

### 开发模式（Console 集成）

```
单进程: console.exe
  ├── HTTP API (:8080)
  ├── 网关管理 (Launcher)
  │   ├── MQTT (:1883)
  │   ├── TCP (:9000)
  │   └── Modbus (:502)
  ├── 规则引擎 (in-process)
  └── SQLite (datacenter.db)

外部依赖:
  ├── NATS (:4222)    — Docker
  ├── TDengine (:6030) — Docker 或本地
  └── Redis (:6379)   — Docker (可选)
```

### 生产模式（微服务拆分）

```
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│ console.exe │  │mqtt-gw.exe  │  │ tcp-gw.exe  │
│ :8080 API   │  │ :1883       │  │ :9000       │
└──────┬──────┘  └──────┬──────┘  └──────┬──────┘
       │               │               │
       ▼               ▼               ▼
┌──────────────────────────────────────────────┐
│              NATS JetStream :4222             │
└──────┬───────────────────────────┬───────────┘
       │                           │
       ▼                           ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│modbus-gw.exe │  │rule-engine   │  │timeseries    │
│ :502         │  │  .exe        │  │  -writer.exe │
└──────────────┘  └──────────────┘  └──────┬───────┘
                                          │
                                          ▼
                                   ┌──────────────┐
                                   │  TDengine    │
                                   │  :6030       │
                                   └──────────────┘
```

---

## 11. 统一消息格式

```json
{
  "device_id": "temp-sensor-001",
  "domain_id": "factory-a",
  "model_id": "temp_sensor_v1",
  "type": 0,
  "timestamp": 1724488800000,
  "qos": 0,
  "units": [
    {
      "topic": "property/temperature",
      "payload": "eyJ2YWx1ZSI6MjUuNn0=",
      "timestamp": 1724488800000,
      "metadata": {}
    }
  ],
  "metadata": {
    "region": "east",
    "device_type": "sensor"
  }
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `device_id` | string | 设备唯一标识 |
| `domain_id` | string | 所属业务域 |
| `model_id` | string | 关联物模型 ID |
| `type` | int | 0=数据, 1=指令, 2=事件, 3=确认 |
| `timestamp` | int64 | Unix 毫秒时间戳 |
| `qos` | int | 0=至多一次, 1=至少一次, 2=恰好一次 |
| `units` | array | 数据单元数组（支持批量上报） |
| `metadata` | map | 扩展元数据 |

---

## 12. 技术栈

| 层 | 技术 |
|---|---|
| 前端 | Vue 3 + Composition API, Pinia, Vue Router, Element Plus, Axios, Echarts |
| HTTP 框架 | Gin |
| ORM | GORM (SQLite) |
| 消息中间件 | NATS JetStream |
| 时序数据库 | TDengine 2.6 (原生 CGO 驱动) |
| 缓存 | Redis 7 |
| JS 脚本引擎 | goja (Go 原生 JS 运行时) |
| 日志 | zerolog |
| 协议 | MQTT 3.1.1, TCP 自定义帧, Modbus TCP (MBAP), OPC-UA |
