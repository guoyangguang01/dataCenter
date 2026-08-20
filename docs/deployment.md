# IoT 数据中台 — 部署文档

---

## 目录

- [环境要求](#环境要求)
- [快速启动（开发环境）](#快速启动开发环境)
- [手动部署](#手动部署)
- [生产部署](#生产部署)
- [配置说明](#配置说明)
- [故障排查](#故障排查)

---

## 环境要求

### 开发环境

| 组件 | 版本要求 | 说明 |
|------|----------|------|
| Go | ≥ 1.22 | 后端编译运行 |
| Node.js | ≥ 18 | 前端构建 |
| Docker | ≥ 20.10 | 基础设施容器（可选） |

### 生产环境

| 组件 | 用途 | 推荐配置 |
|------|------|----------|
| NATS + JetStream | 消息总线 | 单机部署，磁盘持久化 |
| TDengine | 时序存储 | taosAdapter REST API |
| Redis | 缓存/设备状态 | Redis 7+ |
| SQLite | 关系存储 | 开发环境；生产环境建议迁移至 MySQL/PostgreSQL |
| Nginx | 反向代理 + 静态文件 | 代理 API 并托管前端 |

---

## 快速启动（开发环境）

### 1. 启动基础设施

```bash
cd deploy
docker-compose up -d
```

启动 NATS（JetStream）、TDengine、Redis 三个服务。

验证：

```bash
# NATS
curl http://localhost:8222/healthz

# TDengine
curl http://localhost:6041/restful

# Redis
redis-cli ping
```

### 2. 编译所有服务

```bash
make build          # 编译 8 个服务到 bin/
```

### 3. 启动服务

**最小化启动（仅管理控制台）：**

```bash
go run ./cmd/console
```

**全量启动（所有微服务）：**

| 服务 | 命令 | 端口 | 说明 |
|------|------|------|------|
| console | `bin\console.exe` | :8080 | HTTP API + 管理控制台 |
| mqtt-gateway | `bin\mqtt-gateway.exe` | :1883 | MQTT 协议网关 |
| tcp-gateway | `bin\tcp-gateway.exe` | :9000 | TCP 自定义协议网关 |
| modbus-gateway | `bin\modbus-gateway.exe` | :502 | Modbus TCP 工业网关 |
| device-service | `bin\device-service.exe` | — | 设备管理 + Redis 状态 |
| rule-engine | `bin\rule-engine.exe` | — | 规则引擎 + NATS 消费 |
| timeseries-writer | `bin\timeseries-writer.exe` | — | 时序写入 + NATS 消费 |
| alert-service | `bin\alert-service.exe` | — | 告警服务 |

> 首次启动 console 会自动创建 SQLite 数据库并建表。

### 4. 启动前端

```bash
cd web
npm install
npm run dev
```

前端默认监听 `:3000`，API 请求自动代理到 `localhost:8080`。

### 5. 访问

打开浏览器访问 `http://localhost:3000`

---

## 手动部署

### 不使用 Docker

如果没有 Docker，可以直接安装并运行基础设施服务：

#### NATS

```bash
# 下载
wget https://github.com/nats-io/nats-server/releases/latest/download/nats-server-linux-amd64.tar.gz
tar xzf nats-server-linux-amd64.tar.gz

# 启动（启用 JetStream）
./nats-server -js -sd /data/nats -p 4222 -m 8222
```

#### TDengine

```bash
# 参考官方文档安装 TDengine
# taosAdapter 默认监听 6041 端口
systemctl start taosadapter
```

#### Redis

```bash
redis-server --appendonly yes
```

### 后端部署

```bash
# 编译所有服务
make build

# 运行所需的服务
./bin/console              # 管理控制台
./bin/mqtt-gateway         # MQTT 网关
./bin/tcp-gateway          # TCP 网关
./bin/modbus-gateway       # Modbus 网关
./bin/device-service       # 设备服务
./bin/rule-engine          # 规则引擎
./bin/timeseries-writer    # 时序写入
./bin/alert-service        # 告警服务
```

生产环境建议使用 systemd 管理每个服务（见下方 systemd 配置）。

### 前端部署

```bash
cd web
npm run build
# 产物在 web/dist/ 目录
```

将 `web/dist/` 内容复制到 Nginx 静态文件目录：

```bash
cp -r web/dist/* /var/www/datacenter/
```

Nginx 配置示例：

```nginx
server {
    listen 80;
    server_name console.example.com;

    # 前端静态文件
    location / {
        root /var/www/datacenter;
        index index.html;
        try_files $uri $uri/ /index.html;
    }

    # API 反向代理
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

---

## 生产部署

### 1. 数据库迁移

当前开发环境使用 SQLite，生产环境建议迁移至 MySQL 或 PostgreSQL：

修改以下服务中的数据库连接（`cmd/console/main.go`、`cmd/device-service/main.go`、`cmd/rule-engine/main.go`、`cmd/alert-service/main.go`）：

```go
// SQLite（开发）
db, _ = gorm.Open(sqlite.Open("datacenter.db"), &gorm.Config{})

// MySQL（生产）
import "gorm.io/driver/mysql"
db, _ = gorm.Open(mysql.Open("user:pass@tcp(127.0.0.1:3306)/datacenter?parseTime=true"), &gorm.Config{})

// PostgreSQL（生产）
import "gorm.io/driver/postgres"
db, _ = gorm.Open(postgres.Open("host=localhost user=postgres dbname=datacenter sslmode=disable"), &gorm.Config{})
```

### 2. NATS 高可用

单机部署时启用 JetStream 磁盘持久化：

```bash
nats-server -js -sd /data/nats -p 4222 -m 8222
```

Docker Compose 中已配置 `restart: always`，容器崩溃会自动重启。

### 3. 安全配置

- **TLS**：通过 Nginx 做 TLS 终止，后端仅监听内网端口
- **CORS**：修改 `cmd/console/main.go` 中的 `Access-Control-Allow-Origin` 为具体域名
- **设备认证**：设备通过 Token 认证（创建设备时自动生成），网关层验证

### 4. 使用 systemd 管理服务

为每个服务创建 systemd unit 文件，例如 `/etc/systemd/system/datacenter-console.service`：

```ini
[Unit]
Description=DataCenter Console
After=network.target

[Service]
Type=simple
User=datacenter
WorkingDirectory=/opt/datacenter
ExecStart=/opt/datacenter/bin/console
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

其他服务同理，替换 `ExecStart` 路径即可：

| Unit 文件 | ExecStart |
|-----------|-----------|
| datacenter-console.service | `/opt/datacenter/bin/console` |
| datacenter-mqtt-gateway.service | `/opt/datacenter/bin/mqtt-gateway` |
| datacenter-tcp-gateway.service | `/opt/datacenter/bin/tcp-gateway` |
| datacenter-modbus-gateway.service | `/opt/datacenter/bin/modbus-gateway` |
| datacenter-device-service.service | `/opt/datacenter/bin/device-service` |
| datacenter-rule-engine.service | `/opt/datacenter/bin/rule-engine` |
| datacenter-timeseries-writer.service | `/opt/datacenter/bin/timeseries-writer` |
| datacenter-alert-service.service | `/opt/datacenter/bin/alert-service` |

```bash
sudo systemctl daemon-reload
sudo systemctl enable datacenter-*
sudo systemctl start datacenter-console datacenter-mqtt-gateway datacenter-rule-engine datacenter-timeseries-writer
```

---

## 配置说明

### 服务端口

| 服务 | 端口 | 说明 |
|------|------|------|
| Console API | 8080 | HTTP API 服务 |
| MQTT Gateway | 1883 | MQTT 协议接入 |
| TCP Gateway | 9000 | TCP 自定义协议接入 |
| Modbus Gateway | 502 | Modbus TCP 工业协议接入 |
| 前端开发服务器 | 3000 | Vite 开发服务器 |
| NATS 客户端 | 4222 | NATS 消息通信 |
| NATS 监控 | 8222 | NATS HTTP 监控页面 |
| TDengine REST | 6041 | 时序数据库 REST API |
| Redis | 6379 | 缓存服务 |

### 数据库文件

各服务独立使用 SQLite 数据库（开发环境），首次启动自动创建：

| 服务 | 数据库文件 | 包含表 |
|------|-----------|--------|
| console | datacenter.db | devices, domains, domain_members, thing_models, device_model_bindings, rule_configs, webhook_configs, alert_logs |
| device-service | device-service.db | devices |
| rule-engine | rule-engine.db | rule_configs |
| alert-service | alert-service.db | webhook_configs, alert_logs |

> 生产环境建议统一使用 MySQL/PostgreSQL，避免多服务各自维护 SQLite。

### Makefile 命令

```bash
make build              # 编译所有 8 个服务
make test               # 运行测试
make docker-up          # 启动基础设施
make docker-down        # 停止基础设施
make clean              # 清理构建产物
```

---

## 故障排查

### 端口被占用

```bash
# 检查所有服务端口
netstat -ano | findstr ":8080 :1883 :9000 :502"

# 或逐个检查
netstat -ano | findstr :8080   # console
netstat -ano | findstr :1883   # mqtt-gateway
netstat -ano | findstr :9000   # tcp-gateway
netstat -ano | findstr :502    # modbus-gateway
```

### 数据库文件损坏

删除对应服务的数据库文件，重启会自动重新创建：

```bash
rm datacenter.db          # console
rm device-service.db      # device-service
rm rule-engine.db         # rule-engine
rm alert-service.db       # alert-service
```

### NATS 连接失败

确认 NATS 服务正在运行：

```bash
docker ps | grep nats
# 或
curl http://localhost:8222/healthz
```

### 前端 API 请求 404

确认前端 Vite 代理配置正确（`web/vite.config.js`）：

```js
proxy: {
  "/api": { target: "http://localhost:8080", changeOrigin: true }
}
```

确保后端服务已启动且监听在 8080 端口。
