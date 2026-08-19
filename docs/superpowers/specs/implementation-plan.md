# 数据中台实施计划

## Context

基于已完成的设计文档 `docs/superpowers/specs/2026-08-19-data-center-design.md`，按模块依赖关系分阶段实施。

技术栈：Go (网关/服务) + NATS+JetStream (消息) + TDengine (时序) + Redis (缓存) + Vue3 (前端)

---

## 阶段一：基础设施与核心消息层（第1-2周）

### 1.1 项目初始化
- 创建 Go module：`github.com/xxx/datacenter`
- 目录结构：`cmd/` `internal/` `pkg/` `api/` `web/` `deploy/` `proto/`
- Docker Compose：NATS(-js) + TDengine + Redis
- Makefile：build / test / docker-up / proto-gen

### 1.2 统一消息格式（Protobuf）
- 定义 `DeviceEnvelope` / `MessageUnit` / `MessageType` / `QoSLevel`
- 生成 Go 代码
- 编写单元测试：序列化/反序列化、批量 units

### 1.3 NATS 客户端封装
- 封装 JetStream producer / consumer
- Topic 工具函数：构建 topic、解析 topic、通配匹配
- 连接管理：断线重连、健康检查

**交付物**：docker-compose 一键启动、Protobuf 生成代码、NATS 封装库 + 单测

---

## 阶段二：协议网关层（第3-4周）

### 2.1 网关框架
- 实现 `GatewayAdapter` 接口
- 网关注册中心 `GatewayRegistry`
- 配置加载（YAML）
- 启停管理

### 2.2 MQTT 网关
- Transport：TCP 监听、TLS 支持
- Protocol Parser：MQTT 协议解析（CONNECT/PUBLISH/SUBSCRIBE/PINGREQ）
- Codec：MQTT Publish <-> DeviceEnvelope
- Session Manager：在线/离线状态管理
- 认证：Token 验证
- ACL：Redis 查询设备权限

### 2.3 TCP/gRPC 网关
- Transport：TCP 长连接 + TLS
- 自定义协议帧解析（长度前缀 + Protobuf body）
- Codec：自定义帧 <-> DeviceEnvelope
- 心跳帧处理

### 2.4 Modbus 网关
- Modbus TCP 协议解析
- 寄存器地址 -> 数据点位映射表
- 轮询调度器
- Codec：Modbus 响应 <-> DeviceEnvelope

**交付物**：3个可运行的网关 + 集成测试（模拟设备连接）

---

## 阶段三：设备管理层（第5周）

### 3.1 设备数据模型
- Device / DeviceStatus / DeviceCredentials 数据库表
- GORM 模型定义

### 3.2 设备认证
- 设备级凭证验证（Token）
- 域级密钥验证（自动发现模式）
- ACL 权限缓存（Redis）

### 3.3 设备状态管理
- Redis 存储在线/离线状态
- last_seen 更新 + 定时扫描兜底
- 心跳配置（按设备类型）

### 3.4 设备管理 API
- REST API：CRUD、批量操作、状态查询
- JWT 中间件（管理员认证）

**交付物**：设备管理服务 + API + 集成测试

---

## 阶段四：规则引擎层（第6-7周）

### 4.1 规则引擎框架
- 规则链 Pipeline 执行器
- 节点注册机制
- NATS 订阅触发

### 4.2 内建节点
- Filter / Transform / Condition / Aggregate / Action
- Action 类型：publish(下发NATS) / alert(告警) / log(日志)

### 4.3 脚本节点
- Goja JS 引擎集成
- 沙箱隔离（CPU/内存限制）
- 超时中断

### 4.4 状态管理
- 内存窗口缓冲
- JetStream 回放重建

**交付物**：规则引擎服务 + 单测 + 集成测试

---

## 阶段五：物模型与时序存储（第8周）

### 5.1 物模型
- ThingModel / PropertyDef / CommandDef / EventDef 数据模型
- 设备-模型绑定
- 模型管理 API

### 5.2 TDengine 时序存储
- 超表/子表自动创建
- 时序写入服务：NATS consumer -> 内存缓冲 -> 批量写入
- 写入失败：重试 + NATS 未 ACK 保留
- 查询 API：按设备/时间范围/聚合

### 5.3 数据校验
- 物模型驱动的数据校验（类型、范围）
- 异常数据标记

**交付物**：物模型服务 + 时序写入服务 + 查询 API

---

## 阶段六：告警与域管理（第9周）

### 6.1 告警通知
- Webhook 发送器
- 告警去重（5分钟窗口）
- 限流（每分钟 N 条）
- 失败重试（3次退避）
- Webhook 管理 API

### 6.2 业务域隔离
- Domain CRUD API
- 域隔离中间件（自动注入 domain_id）
- 域成员管理 + 角色权限
- TDengine 分库策略

**交付物**：告警服务 + 域管理服务 + 中间件

---

## 阶段七：管理控制台（第10-12周）

### 7.1 前端框架
- Vue 3 + Element Plus + Vite
- 路由、状态管理、JWT 认证
- 域切换器（顶部）

### 7.2 核心页面
- 设备管理：列表、注册、详情、状态
- 数据面板：实时图表（WebSocket）、历史查询、导出
- 规则引擎：规则列表、可视化配置、运行日志
- 物模型管理：模型定义、属性/命令/事件编辑
- 域管理：域列表、成员管理、权限配置
- 告警中心：告警列表、Webhook 配置
- 系统监控：网关状态、连接数、消息流量

**交付物**：完整的管理控制台

---

## 阶段八：集成测试与优化（第13周）

- 端到端测试：模拟设备 -> 网关 -> NATS -> 规则引擎 -> 告警/存储
- 性能测试：千级设备并发连接 + 消息吞吐
- 稳定性测试：长时间运行、异常恢复
- 文档完善：API 文档、部署文档、用户手册

---

## 目录结构规划

```
dataCenter/
├── cmd/
│   ├── mqtt-gateway/        # MQTT 网关入口
│   ├── tcp-gateway/         # TCP 网关入口
│   ├── modbus-gateway/      # Modbus 网关入口
│   ├── device-service/      # 设备管理服务
│   ├── rule-engine/         # 规则引擎
│   ├── timeseries-writer/   # 时序写入服务
│   ├── alert-service/       # 告警服务
│   └── console/             # 控制台后端
├── internal/
│   ├── gateway/             # 网关框架（Adapter, Registry, Codec）
│   ├── mqtt/                # MQTT 协议实现
│   ├── tcp/                 # TCP 协议实现
│   ├── modbus/              # Modbus 协议实现
│   ├── message/             # DeviceEnvelope 模型
│   ├── device/              # 设备管理逻辑
│   ├── rule/                # 规则引擎逻辑
│   ├── model/               # 物模型逻辑
│   ├── timeseries/          # 时序存储逻辑
│   ├── alert/               # 告警逻辑
│   ├── domain/              # 域管理逻辑
│   └── auth/                # 认证鉴权
├── pkg/
│   ├── nats/                # NATS 客户端封装
│   ├── redis/               # Redis 客户端封装
│   ├── tdengine/            # TDengine 客户端封装
│   └── util/                # 通用工具
├── api/
│   └── v1/                  # REST API 定义
├── proto/                   # Protobuf 定义
├── web/                     # Vue 3 前端
├── deploy/
│   ├── docker-compose.yml
│   └── Dockerfile.*
├── docs/
│   └── superpowers/
│       └── specs/           # 设计文档
├── Makefile
├── go.mod
└── README.md
```
