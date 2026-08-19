# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

IoT 数据中台 (IoT Data Center) — a Go microservices platform for multi-protocol device connectivity (MQTT, TCP, Modbus) with real-time data ingestion, rule engine processing, time-series storage, and a Vue 3 web management console.

## Build & Run Commands

### Backend (Go)

```bash
make build                  # Build all 8 service binaries into bin/
make build-mqtt-gateway     # Build a single service
make test                   # Run all tests (go test ./... -v -count=1)
make test-cover             # Run tests with coverage report (coverage.html)
make proto-gen              # Regenerate Go code from proto/message.proto
make docker-up              # Start infrastructure (NATS, TDengine, Redis)
make docker-down            # Stop infrastructure
make clean                  # Remove bin/, coverage.out, coverage.html
```

### Frontend (Vue 3)

```bash
cd web
npm install                 # Install dependencies
npm run dev                 # Vite dev server on :3000, proxies /api → localhost:8080
npm run build               # Production build
```

### Quick Start

```bash
cd deploy && docker-compose up -d    # Start NATS, TDengine, Redis
cd .. && make build                  # Build all services
cd web && npm install && npm run dev # Start frontend
```

## Architecture

### Microservice Decomposition (8 services)

Each service has an entry point under `cmd/<service>/main.go`. Currently only the `console` service has a wired HTTP server; the rest are stubs. Core business logic lives in `internal/<package>/`.

| Service | Package | Role |
|---|---|---|
| mqtt-gateway | `internal/mqtt/` | MQTT 3.1.1 protocol gateway with custom packet parser |
| tcp-gateway | `internal/tcp/` | TCP binary framing protocol gateway (`[4B len][2B type][payload]`) |
| modbus-gateway | `internal/modbus/` | Modbus TCP (MBAP header) industrial protocol gateway |
| device-service | `internal/device/` | Device CRUD, auth, online/offline status tracking |
| rule-engine | `internal/rule/` | Pipeline-based data processing (filter/transform/condition/action/script) |
| timeseries-writer | `internal/timeseries/` | Batched TDengine writer via REST API |
| alert-service | `internal/alert/` | Webhook-based alert delivery with dedup and rate limiting |
| console | `api/v1/` + `cmd/console` | HTTP API server (Gin framework) |

### Message Flow

```
Device → Gateway (MQTT/TCP/Modbus) → NATS JetStream → Rule Engine → Timeseries Writer (TDengine)
                                                           ↓
                                                     Alert Service (webhooks)
```

### Gateway Layer (`internal/gateway/`)

- **`GatewayAdapter` interface:** `Start()`, `Stop()`, `OnDeviceStatusChanged()` — every protocol gateway implements this
- **`Publisher` interface:** `PublishEnvelope()` — gateways publish to NATS via this
- **`NATSPublisher`:** Serializes `DeviceEnvelope` to JSON and publishes to JetStream

### NATS Topic Convention (`pkg/nats/`)

Six-level topic hierarchy: `domains.{domain_id}.devices.{region}.{device_type}.{device_id}.{direction}`
- Direction: `up` (device report) or `down` (command delivery)
- Wildcards: `>` (multi-segment), `*` (single segment)
- JetStream streams: `DEVICE_DATA` (7-day), `DEVICE_COMMAND` (3-day), `SYSTEM_EVENTS` (30-day)

### Unified Message Format (`proto/message.proto`, `internal/message/`)

`DeviceEnvelope` is the top-level wrapper: device_id, domain_id, model_id, timestamp, type (DATA/COMMAND/EVENT/ACK), units array, QoS level, metadata. Each `MessageUnit` contains topic, payload bytes, timestamp, metadata.

### Rule Engine (`internal/rule/`)

Pipeline pattern: Data source → Filter → Transform → Condition → Action. Each node implements `Execute(ctx, envelope, state)`. The `ScriptNode` runs JavaScript via Goja VM (5s timeout). `PipelineState` provides thread-safe state and sliding window aggregation.

### Device Auth (`internal/device/`)

Two-phase authentication: device token first, then domain key (`domain_key:domain_id` format). AuthManager caches permissions in Redis (5-min TTL). StatusManager tracks online/offline via Redis with a 3-minute offline threshold.

### Web Frontend (`web/src/`)

Vue 3 + Composition API (`<script setup>`), Pinia, Vue Router (HTML5 history), Element Plus UI, Axios. Currently only the Devices view has full CRUD; Dashboard uses hardcoded sample data; Rules/Models/Domains/Alerts are placeholders.

## Key Conventions

- **Internal packages** (`internal/`) are not importable externally — standard Go project layout
- **Shared libraries** (`pkg/nats/`) are reusable across services
- **Protobuf** definitions in `proto/` — run `make proto-gen` after changes
- **GORM** for relational DB models (MySQL/PostgreSQL compatible)
- **TDengine** accessed via REST API, not native driver
- **No linting or formatting config** exists — follow standard Go and Vue conventions
- **No TypeScript** — frontend is plain JavaScript (`.js` + `.vue` files)
