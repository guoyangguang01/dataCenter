# IoT 数据中台

支持 MQTT/TCP/Modbus 多协议接入的实时数据交互平台。

## 技术栈

- Go 1.26 + NATS + TDengine + Redis
- Vue 3 + Element Plus

## 快速开始

1. cd deploy && docker-compose up -d
2. make build
3. cd web && npm install && npm run dev

## API

- POST /api/v1/devices
- GET /api/v1/devices
- GET /api/v1/devices/:id
- PUT /api/v1/devices/:id
- DELETE /api/v1/devices/:id
