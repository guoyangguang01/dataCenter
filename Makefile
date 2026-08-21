# Variables
PROJECT_NAME := datacenter
BUILD_DIR := bin
PROTOS := proto/*.proto

# Go build tags
GO := go
GOFLAGS := -v

# Docker
COMPOSE_FILE := deploy/docker-compose.yml

.PHONY: all build clean test docker-up docker-down proto-gen

all: build

# Build all services
build:
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/mqtt-gateway.exe ./cmd/mqtt-gateway
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/tcp-gateway.exe ./cmd/tcp-gateway
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/modbus-gateway.exe ./cmd/modbus-gateway
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/opcua-gateway.exe ./cmd/opcua-gateway
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/device-service.exe ./cmd/device-service
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/rule-engine.exe ./cmd/rule-engine
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/timeseries-writer.exe ./cmd/timeseries-writer
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/alert-service.exe ./cmd/alert-service
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/console.exe ./cmd/console

# Build a specific service
build-%:
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$*.exe ./cmd/$*

# Run tests
test:
	$(GO) test ./... -v -count=1

# Run tests with coverage
test-cover:
	$(GO) test ./... -v -count=1 -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html

# Docker
docker-up:
	docker compose -f $(COMPOSE_FILE) up -d

docker-down:
	docker compose -f $(COMPOSE_FILE) down

docker-logs:
	docker compose -f $(COMPOSE_FILE) logs -f

# Protobuf generation
proto-gen:
	protoc --go_out=. --go_opt=paths=source_relative proto/*.proto

# Clean
clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html

# Simulator commands
sim-mqtt:               ## 启动 MQTT 模拟器
	$(GO) run cmd/simulator/main.go --protocol mqtt --config configs/simulator/scenarios/mqtt_sim.yaml

sim-tcp-client:         ## 启动 TCP 客户端模拟器
	$(GO) run cmd/simulator/main.go --protocol tcp --mode client --config configs/simulator/scenarios/tcp_sim.yaml

sim-tcp-server:         ## 启动 TCP 服务端模拟器
	$(GO) run cmd/simulator/main.go --protocol tcp --mode server --config configs/simulator/scenarios/tcp_sim.yaml

sim-modbus-slave:       ## 启动 Modbus 从站模拟器
	$(GO) run cmd/simulator/main.go --protocol modbus --mode slave --config configs/simulator/scenarios/modbus_sim.yaml

sim-modbus-master:      ## 启动 Modbus 主站模拟器
	$(GO) run cmd/simulator/main.go --protocol modbus --mode master --config configs/simulator/scenarios/modbus_sim.yaml

sim-opcua:              ## 启动 OPC UA 模拟器
	$(GO) run cmd/simulator/main.go --protocol opcua --config configs/simulator/scenarios/opcua_sim.yaml

sim-stop:               ## 停止所有模拟器
	pkill -f "cmd/simulator" || true
