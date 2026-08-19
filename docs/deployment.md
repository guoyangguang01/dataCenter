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

### 2. 编译后端

```bash
# 从项目根目录
go build ./cmd/console
```

或使用 Makefile：

```bash
make build
```

### 3. 启动后端

```bash
./console           # Linux/Mac
console.exe         # Windows
```

或开发模式：

```bash
go run ./cmd/console
```

后端默认监听 `:8080`，启动时自动创建 SQLite 数据库文件 `datacenter.db` 并建表。

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
# 编译
CGO_ENABLED=1 go build -o bin/console ./cmd/console

# 运行（使用 systemd 管理）
./bin/console
```

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

修改 `cmd/console/main.go` 中的数据库连接：

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

### 4. 使用 systemd 管理后端

创建 `/etc/systemd/system/datacenter-console.service`：

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

```bash
sudo systemctl daemon-reload
sudo systemctl enable datacenter-console
sudo systemctl start datacenter-console
```

---

## 配置说明

### 服务端口

| 服务 | 端口 | 说明 |
|------|------|------|
| Console API | 8080 | HTTP API 服务 |
| 前端开发服务器 | 3000 | Vite 开发服务器 |
| NATS 客户端 | 4222 | NATS 消息通信 |
| NATS 监控 | 8222 | NATS HTTP 监控页面 |
| TDengine REST | 6041 | 时序数据库 REST API |
| Redis | 6379 | 缓存服务 |

### 数据库表

启动时自动创建以下表：

| 表名 | 所属模块 | 说明 |
|------|----------|------|
| devices | 设备管理 | 设备档案 |
| domains | 业务域 | 域定义 |
| domain_members | 业务域 | 域成员与角色 |
| thing_models | 物模型 | 模型定义 |
| device_model_bindings | 物模型 | 设备-模型绑定 |
| rule_configs | 规则引擎 | 规则持久化 |
| webhook_configs | 告警 | Webhook 配置 |
| alert_logs | 告警 | 告警发送日志 |

### Makefile 命令

```bash
make build              # 编译所有服务
make build-console      # 仅编译 Console
make test               # 运行测试
make docker-up          # 启动基础设施
make docker-down        # 停止基础设施
make clean              # 清理构建产物
```

---

## 故障排查

### 端口被占用

```bash
# Windows
netstat -ano | findstr :8080

# Linux
lsof -i :8080
```

更换端口：修改 `cmd/console/main.go` 中 `Addr: ":8080"`。

### 数据库文件损坏

删除 `datacenter.db`，重启服务会自动重新创建：

```bash
rm datacenter.db
go run ./cmd/console
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
