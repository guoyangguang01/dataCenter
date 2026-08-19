# IoT 数据中台

多协议设备接入、实时数据处理、时序存储与可视化管理的一体化平台。

## 功能特性

- **多协议网关** — MQTT 3.1.1 / TCP 自定义帧 / Modbus TCP 三种协议同时接入
- **设备管理** — 设备注册、认证、在线/离线状态跟踪
- **物模型** — 属性/命令/事件三层结构定义设备数据模型
- **规则引擎** — Pipeline 模式数据处理：过滤、转换、条件、聚合、JavaScript 脚本
- **时序存储** — TDengine 存储设备上报数据，支持按时间/设备/聚合查询
- **告警通知** — Webhook 通知 + 5 分钟去重 + 每分钟限流 + 3 次退避重试
- **业务域隔离** — 多租户隔离，域级别设备/模型/规则/告警独立
- **管理控制台** — Vue 3 + Element Plus Web 界面，6 个功能模块

## 架构

```
Device → Gateway (MQTT/TCP/Modbus) → NATS JetStream → Rule Engine → Timeseries Writer (TDengine)
                                                          ↓
                                                    Alert Service (Webhooks)
```

### 微服务分解（8 个服务）

| 服务 | 端口 | 职责 |
|------|------|------|
| console | 8080 | HTTP API + 管理控制台后端 |
| mqtt-gateway | 1883 | MQTT 协议网关 |
| tcp-gateway | 9000 | TCP 自定义协议网关 |
| modbus-gateway | 502 | Modbus TCP 工业协议网关 |
| device-service | — | 设备管理服务 |
| rule-engine | — | 规则引擎服务 |
| timeseries-writer | — | TDengine 时序写入服务 |
| alert-service | — | 告警通知服务 |

> 所有 8 个服务入口均已接线，可独立编译运行。

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

### 1. 启动基础设施

```bash
cd deploy
docker-compose up -d
```

启动 NATS（JetStream）、TDengine、Redis。

### 2. 编译并启动后端

```bash
# 从项目根目录
go build ./cmd/console
./console          # Linux/Mac
console.exe        # Windows
```

或直接：

```bash
go run ./cmd/console
```

后端监听 `http://localhost:8080`，首次启动自动创建数据库并建表。

### 3. 启动前端

```bash
cd web
npm install
npm run dev
```

前端监听 `http://localhost:3000`，API 自动代理到后端。

### 4. 访问控制台

打开浏览器访问 **http://localhost:3000**

### 一键启动（Makefile）

```bash
cd deploy && docker-compose up -d
make build
make run-console
cd web && npm install && npm run dev
```

## 项目结构

```
dataCenter/
├── cmd/                        # 8 个微服务入口
│   ├── console/                # HTTP API 服务（已接线）
│   ├── mqtt-gateway/
│   ├── tcp-gateway/
│   ├── modbus-gateway/
│   ├── device-service/
│   ├── rule-engine/
│   ├── timeseries-writer/
│   └── alert-service/
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
│   └── message/                # 统一消息格式
├── pkg/nats/                   # NATS 客户端封装
├── proto/                      # Protobuf 定义
├── web/                        # Vue 3 前端
│   └── src/
│       ├── views/              # 6 个页面
│       ├── api/                # API 客户端
│       ├── stores/             # Pinia 状态
│       └── router/             # 路由配置
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

```bash
cd deploy && docker-compose up -d     # 启动
cd deploy && docker-compose down      # 停止
```

## 测试

```bash
go test ./... -v -count=1            # 运行所有测试
go test ./internal/message/ -v       # 消息格式测试
go test ./pkg/nats/ -v               # NATS Topic 测试
go test ./tests/ -v                  # 集成测试
```

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

- 其余 7 个 `cmd/` 服务入口接线
- 端到端集成测试（模拟设备→网关→NATS→规则→存储）
- 前端编辑功能（设备编辑、规则可视化编辑器）
- 生产环境数据库迁移（MySQL / PostgreSQL）
- 可观测性体系（Metrics / Tracing / Logging）

## License

MIT
