# 数据中台设计方案

## Context

设计一个 IoT 数据中台，支持多种不同协议的客户端（MQTT、gRPC/TCP 私有协议、Modbus/CoAP 工业协议）之间进行实时数据交互。

核心需求：设备间通信（经平台中转）、时序数据持久化、<=10K 设备规模。

---

## 一、架构方案：协议网关 + 统一消息总线

技术栈：

| 层 | 技术选型 | 说明 |
|---|---|---|
| 协议网关 | Go | 高并发、单二进制部署、跨平台 |
| 消息总线 | NATS + JetStream | 轻量、高性能、原生 topic 路由 |
| 规则引擎 | Go + Goja(JS) | 配置驱动 + 脚本扩展 |
| 设备管理 | Go + Redis | 认证凭证缓存、设备状态 |
| 时序存储 | TDengine (Docker) | 高性能时序数据库 |
| 管理控制台 | Vue 3 + Element Plus | 成熟的管理后台方案 |
| 通信协议 | Protobuf | 统一消息序列化 |
| 用户认证 | JWT | 标准方案 |

---

## 二、统一消息格式

核心结构：

```protobuf
message DeviceEnvelope {
  string device_id = 1;
  string domain_id = 2;
  string model_id = 3;
  int64 timestamp = 4;
  MessageType type = 5;          // DATA / COMMAND / EVENT / ACK
  repeated MessageUnit units = 6;
  QoSLevel qos = 7;
  map<string,string> metadata = 8;
}

message MessageUnit {
  string topic = 1;
  bytes payload = 2;
  int64 timestamp = 3;
  map<string,string> metadata = 4;
}

enum MessageType { DATA = 0; COMMAND = 1; EVENT = 2; ACK = 3; }
enum QoSLevel { AT_MOST_ONCE = 0; AT_LEAST_ONCE = 1; EXACTLY_ONCE = 2; }
```

消息大小限制（H1）：实时通道 512KB 上限，大文件走对象存储

---

## 三、协议网关层

统一适配接口：

```go
type GatewayAdapter interface {
    Start(config GatewayConfig) error
    OnMessageReceived(envelope *DeviceEnvelope) error
    DeliverMessage(envelope *DeviceEnvelope) error
    OnDeviceStatusChanged(deviceID string, status DeviceStatus) error
    Stop() error
}
```

网关内部结构：Transport(网络+TLS终止) -> Protocol Parser(协议解析+认证) -> Codec(双向翻译) -> Session Manager(状态管理)

可插拔注册机制，新增协议只需实现接口+配置文件。

---

## 四、消息总线层

Topic 规范（6级）：domains.{domain_id}.devices.{region}.{device_type}.{device_id}.{direction}

JetStream Streams：
- DEVICE_DATA：设备上报，7天
- DEVICE_COMMAND：设备下发，WorkQueue，3天
- SYSTEM_EVENTS：系统事件，30天

高可用（H4）：单机 + JetStream 磁盘持久化 + restart:always

---

## 五、规则引擎层

规则链 Pipeline：数据源 -> 过滤器 -> 转换器 -> 条件判断 -> 动作

节点类型：Filter / Transform / Script(Goja) / Condition / Aggregate / Action

状态管理（H5）：内存存储，重启后 JetStream 回放重建

---

## 六、设备管理层

接入模式：先注册后接入（核心设备） + 自动发现（临时设备）

心跳检测（H2）：协议层原生机制 + 平台层 last_seen 兜底（默认180s）

离线消息（H3）：不做会话恢复，靠 JetStream 天然缓存（3天）

---

## 七、物模型

三层结构：属性(Property) + 命令(Command) + 事件(Event)

能力：数据校验、字段归一、面板自动生成、规则引擎属性选择

---

## 八、业务域隔离

Topic 前缀隔离 + TDengine 分库 + 控制台域切换 + 后端中间件注入

角色：超级管理员 / 域管理员 / 域操作员 / 域观察者

---

## 九、时序存储层

TDengine 超表+子表，批量写入（1000条或1秒）

写入失败兜底（H7）：内存缓冲(50000条) + NATS未ACK保留(7天) + 设备端补报

---

## 十、告警通知（H8）

Webhook（HTTP POST），适配钉钉/企微/飞书/自研

去重：5分钟窗口内同设备同规则只发一次

限流：每分钟最大N条，失败3次退避重试

---

## 十一、传输层安全（H9）

TLS 终止在网关层：MQTT(8883) / gRPC / WebSocket(8443)

Modbus：VLAN 网络隔离 + 设备白名单（协议不支持TLS）

TCP 私有：Transport 层套 TLS

---

## 十二、管理控制台

模块：设备管理、数据面板、规则引擎、系统监控、物模型管理、域管理、告警中心

---

## 十三、部署方案（开发环境）

Docker Compose：NATS(-js) + TDengine + Redis，全部 restart:always

---

## 未决中风险项（实施阶段按需细化）

payload序列化格式、消息去重、认证失败策略、Modbus轮询细节、消息顺序性、死信队列、规则优先级、规则版本管理、设备影子、降采样策略、操作审计日志、生产部署方案、可观测性体系、标签查询优化、跨域查询、容量规划
