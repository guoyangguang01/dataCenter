# IoT 数据中台 — API 文档

> 基础地址：`http://localhost:8080/api/v1`
> 内容类型：`application/json`

---

## 1. 设备管理

### 1.1 创建设备

```
POST /devices
```

**请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | ✅ | 设备唯一标识 |
| name | string | ✅ | 设备名称 |
| device_type | string | ✅ | 设备类型（如 sensor、actuator） |
| protocol | string | ✅ | 协议（mqtt / tcp / modbus） |
| domain_id | string | ✅ | 所属业务域 |
| region | string | | 地区（如 cn-east） |
| model_id | string | | 绑定的物模型 ID |
| firmware | string | | 固件版本 |

**响应 201：**

```json
{
  "id": "dev-001",
  "name": "温度传感器",
  "device_type": "sensor",
  "protocol": "mqtt",
  "domain_id": "factory-a",
  "token": "6ca64caf...",
  "online": false,
  "status": 0,
  "created_at": "2026-08-19T16:55:19Z"
}
```

> 注：`token` 由服务端自动生成（64 位十六进制），用于设备认证。

---

### 1.2 查询设备列表

```
GET /devices?page=1&page_size=20&domain_id=factory-a&device_type=sensor&protocol=mqtt
```

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | ✅ | 页码，从 1 开始 |
| page_size | int | ✅ | 每页条数，1-100 |
| domain_id | string | | 按业务域过滤 |
| device_type | string | | 按设备类型过滤 |
| protocol | string | | 按协议过滤 |

**响应 200：**

```json
{
  "data": [
    {
      "id": "dev-001",
      "name": "温度传感器",
      "device_type": "sensor",
      "protocol": "mqtt",
      "domain_id": "factory-a",
      "online": false,
      "token": "6ca64caf..."
    }
  ],
  "total": 1,
  "page": 1,
  "size": 20
}
```

---

### 1.3 查询单个设备

```
GET /devices/:id
```

**响应 200：** 返回完整设备对象。
**响应 404：** `{"error": "device not found"}`

---

### 1.4 更新设备

```
PUT /devices/:id
```

**请求体（均为可选）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| name | string | 设备名称 |
| device_type | string | 设备类型 |
| region | string | 地区 |
| model_id | string | 物模型 ID |
| firmware | string | 固件版本 |

**响应 200：** 返回更新后的完整设备对象。

---

### 1.5 删除设备

```
DELETE /devices/:id
```

**响应 200：** `{"message": "deleted"}`

---

### 1.6 验证设备 Token

```
POST /devices/:id/verify
```

**请求体：**

```json
{
  "token": "6ca64caf..."
}
```

**响应 200：**

```json
{
  "device": { ... },
  "verified": true
}
```

**响应 401：** `{"error": "invalid token"}`

---

## 2. 业务域管理

### 2.1 创建域

```
POST /domains
```

**请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | ✅ | 域唯一标识 |
| name | string | ✅ | 域名称 |
| description | string | | 描述 |

**响应 201：** 返回域对象。

---

### 2.2 查询域列表

```
GET /domains
```

**响应 200：**

```json
{
  "data": [
    {
      "id": "factory-a",
      "name": "工厂A",
      "description": "华东工厂",
      "created_at": "2026-08-19T17:35:29Z"
    }
  ]
}
```

---

### 2.3 查询单个域

```
GET /domains/:id
```

---

### 2.4 删除域

```
DELETE /domains/:id
```

---

### 2.5 添加域成员

```
POST /domains/:id/members
```

**请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| user_id | string | ✅ | 用户 ID |
| role | string | ✅ | 角色：`super_admin` / `admin` / `operator` / `viewer` |

**响应 201：** `{"message": "member added"}`

---

### 2.6 查询域成员列表

```
GET /domains/:id/members
```

**响应 200：**

```json
{
  "data": [
    {
      "id": 1,
      "domain_id": "factory-a",
      "user_id": "admin-01",
      "role": "admin",
      "created_at": "2026-08-19T17:35:29Z"
    }
  ]
}
```

---

### 2.7 移除域成员

```
DELETE /domains/:id/members/:userId
```

**响应 200：** `{"message": "member removed"}`

---

## 3. 物模型管理

### 3.1 创建物模型

```
POST /models
```

**请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | ✅ | 模型唯一标识 |
| name | string | ✅ | 模型名称 |
| domain_id | string | ✅ | 所属业务域 |
| properties | array | | 属性定义列表 |
| commands | array | | 命令定义列表 |
| events | array | | 事件定义列表 |

**属性定义（PropertyDef）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 属性 ID |
| name | string | 属性名称 |
| data_type | string | 数据类型：float / int / string / bool / enum |
| unit | string | 单位（如 °C、%、Pa） |
| range | [float, float] | 合法值范围 [最小值, 最大值] |
| required | bool | 是否必填 |
| access_mode | string | 访问模式：r（只读）/ rw（读写） |

**请求示例：**

```json
{
  "id": "temp-sensor",
  "name": "温度传感器",
  "domain_id": "factory-a",
  "properties": [
    {
      "id": "temperature",
      "name": "温度",
      "data_type": "float",
      "unit": "°C",
      "range": [0, 100],
      "required": true,
      "access_mode": "r"
    }
  ]
}
```

---

### 3.2 按域查询模型列表

```
GET /models?domain_id=factory-a
```

---

### 3.3 查询单个模型

```
GET /models/:id
```

---

### 3.4 删除模型

```
DELETE /models/:id
```

---

### 3.5 绑定设备到模型

```
POST /models/bind
```

**请求体：**

```json
{
  "device_id": "dev-001",
  "model_id": "temp-sensor"
}
```

---

### 3.6 解绑设备

```
DELETE /models/unbind/:deviceId
```

---

### 3.7 查询设备绑定的模型

```
GET /models/device/:deviceId
```

---

## 4. 规则引擎

### 4.1 创建规则

```
POST /rules
```

**请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | ✅ | 规则名称 |
| domain_id | string | ✅ | 所属业务域 |
| topic | string | | 订阅的 NATS Subject |
| chain | array | ✅ | 节点链配置 |
| enabled | bool | | 是否启用（默认 false） |

**节点配置（NodeConfig）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 节点 ID |
| type | string | 节点类型：filter / transform / condition / aggregate / script |
| config | object | 节点专属配置 |

**节点类型配置说明：**

| 类型 | 配置字段 |
|------|----------|
| filter | `field`（匹配字段）、`operator`（eq/contains/prefix）、`value`（匹配值） |
| condition | `expression`（条件表达式） |
| aggregate | `window_size`（窗口大小）、`function`（avg/sum/min/max/count） |
| script | `script`（JavaScript 脚本） |

**请求示例：**

```json
{
  "name": "高温告警",
  "domain_id": "factory-a",
  "topic": "sensors.temperature",
  "chain": [
    {
      "id": "f1",
      "type": "filter",
      "config": {
        "field": "topic",
        "operator": "contains",
        "value": "temperature"
      }
    }
  ],
  "enabled": true
}
```

---

### 4.2 查询规则列表

```
GET /rules?domain_id=factory-a
```

---

### 4.3 查询单个规则

```
GET /rules/:id
```

---

### 4.4 更新规则

```
PUT /rules/:id
```

**请求体（均为可选）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| name | string | 规则名称 |
| topic | string | NATS Subject |
| chain | array | 节点链配置 |
| enabled | bool | 是否启用 |

---

### 4.5 删除规则

```
DELETE /rules/:id
```

---

### 4.6 切换规则启用状态

```
PUT /rules/:id/toggle
```

**响应 200：** 返回规则对象，`enabled` 字段取反。

---

## 5. 告警管理

### 5.1 创建 Webhook

```
POST /alerts/webhooks
```

**请求体：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | ✅ | Webhook 名称 |
| domain_id | string | ✅ | 所属业务域 |
| url | string | ✅ | Webhook URL |
| method | string | | 请求方法（默认 POST） |
| headers | object | | 自定义请求头 |
| filter | object | | 过滤条件 |
| rate_limit | object | | 限流配置 |

**过滤条件（WebhookFilter）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| levels | array | 接收的告警级别：critical / warning / info |
| device_types | array | 接收的设备类型 |

**限流配置（RateLimitConfig）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| max_per_minute | int | 每分钟最大请求数（默认 10） |
| dedup_window | int | 去重窗口，秒（默认 300） |

---

### 5.2 查询 Webhook 列表

```
GET /alerts/webhooks?domain_id=factory-a
```

---

### 5.3 查询单个 Webhook

```
GET /alerts/webhooks/:id
```

---

### 5.4 更新 Webhook

```
PUT /alerts/webhooks/:id
```

**请求体：** 同创建。

---

### 5.5 删除 Webhook

```
DELETE /alerts/webhooks/:id
```

---

### 5.6 测试 Webhook

```
POST /alerts/webhooks/:id/test
```

发送一条测试告警事件到指定 Webhook。

**响应 200：** `{"message": "test sent"}`

---

### 5.7 查询告警日志

```
GET /alerts/logs?webhook_id=wh_xxx
```

**响应 200：**

```json
{
  "data": [
    {
      "id": 1,
      "alert_id": "alert_001",
      "webhook_id": "wh_xxx",
      "status": "sent",
      "response": "",
      "created_at": "2026-08-19T17:36:57Z"
    }
  ]
}
```

`status` 取值：`sent`（成功）、`failed`（失败）、`rate_limited`（限流）。

---

## 通用错误响应

| HTTP 状态码 | 场景 |
|-------------|------|
| 400 | 请求参数错误 |
| 401 | 认证失败（如 Token 无效） |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

**错误响应格式：**

```json
{
  "error": "错误描述信息"
}
```
