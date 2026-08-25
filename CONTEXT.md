# Context: IoT Data Center

## Core Concepts

### Device
A physical IoT device (sensor, actuator, controller) that connects to the platform through a protocol gateway. Each device belongs to exactly one Domain and is identified by a `device_id`. Devices have a Device Model that defines their capabilities.

### Device Model
A schema defining what a device can do: which data points it reports (telemetry), which commands it accepts, and its properties. Acts as the "type" for Device instances.

### Domain
An organizational boundary that groups devices, users, and resources. A device belongs to exactly one domain. Domain identity is carried in the `domain_id` field and authenticated via `domain_key`.

### Gateway
A protocol-specific entry point that translates between a wire protocol (MQTT, TCP, Modbus) and the internal message bus (NATS). Gateways handle device connections, authentication, and bidirectional message translation.

### Message (DeviceEnvelope)
The canonical internal representation of any device communication. Wraps one or more MessageUnits. Carries metadata: `device_id`, `domain_id`, `model_id`, `timestamp`, `type`, `qos`.

### MessageUnit
An individual payload within a DeviceEnvelope. Contains `topic`, `payload` (bytes), `timestamp`, and metadata. Multiple units can be batched in a single envelope.

## Data Flow Directions

### Uplink (Device → Cloud)
Device-initiated communication. The device reports telemetry data, events, or acknowledgments through a gateway. The gateway publishes a `DeviceEnvelope` with `type=DATA|EVENT|ACK` to the message bus.

- **Data**: Telemetry readings (temperature, humidity, status). Type = `DATA` (0).
- **Event**: Irregular occurrences (alarm triggered, boot completed). Type = `EVENT` (2).
- **Ack**: Device confirmation of a previously received command. Type = `ACK` (3).

### Downlink (Cloud → Device)
Cloud-initiated communication. A command is published to the message bus, routed through a gateway, and delivered to the target device. The gateway translates the command into the device's wire protocol.

- **Command**: An instruction sent to a device (set value, trigger action, configure). Type = `COMMAND` (1).

## Protocol-Specific Terms

### MQTT Gateway
Handles MQTT 3.1.1 connections. Devices connect via TCP, subscribe/publish to MQTT topics. Downlink commands are delivered as PUBLISH packets to the device's subscribed topic.

### TCP Gateway
Handles raw TCP connections with a binary framing protocol: `[4B length][2B type][payload]`. Frame types: Data (0x0001), Command (0x0002), Heartbeat (0x0003).

### Modbus Gateway
Handles Modbus TCP (MBAP header). Devices are polled for register values (uplink). Downlink writes values to device registers (write single register, write multiple registers).

## Infrastructure Terms

### NATS JetStream
The internal message bus. Provides durable, at-least-once delivery. Topics follow the convention: `domains.{domain_id}.devices.{region}.{device_type}.{device_id}.{direction}`.

### Direction
- `up`: Device-to-cloud (uplink) — stored in `DEVICE_DATA` stream (7-day retention).
- `down`: Cloud-to-device (downlink) — stored in `DEVICE_COMMAND` stream (workqueue, 3-day retention).

### Rule Engine
A pipeline processor that subscribes to uplink data, applies filter/transform/condition/aggregate logic, and triggers actions (alerts, further publishes). Operates on the cloud side only.

### Timeseries Writer
Subscribes to uplink data and batch-writes to TDengine for long-term storage and querying.

### Alert Service
Subscribes to rule engine alert outputs and delivers notifications via webhooks.
