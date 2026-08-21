package mqtt

import (
	"context"
	"encoding/json"
	"fmt"

	mqttlib "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog"
)

// Adapter implements the MQTT protocol adapter
type Adapter struct {
	broker         string
	username       string
	password       string
	clientIDPrefix string
	topicFormat    string
	qos            byte
	cleanSession   bool
	clients        map[string]mqttlib.Client // deviceID -> client
	logger         zerolog.Logger
}

// NewAdapter creates a new MQTT adapter
func NewAdapter(
	broker, username, password, clientIDPrefix, topicFormat string,
	qos byte,
	cleanSession bool,
	logger zerolog.Logger,
) *Adapter {
	return &Adapter{
		broker:         broker,
		username:       username,
		password:       password,
		clientIDPrefix: clientIDPrefix,
		topicFormat:    topicFormat,
		qos:            qos,
		cleanSession:   cleanSession,
		clients:        make(map[string]mqttlib.Client),
		logger:         logger,
	}
}

// Start starts the MQTT adapter (connections are created per-device in SendData)
func (a *Adapter) Start(ctx context.Context) error {
	a.logger.Info().
		Str("broker", a.broker).
		Msg("MQTT adapter ready, connections will be created per device")
	return nil
}

// Stop stops the MQTT adapter
func (a *Adapter) Stop() error {
	for deviceID, client := range a.clients {
		if client.IsConnected() {
			client.Disconnect(1000)
			a.logger.Info().Str("device_id", deviceID).Msg("Disconnected from MQTT broker")
		}
	}
	a.clients = make(map[string]mqttlib.Client)
	return nil
}

// SendData sends data to the MQTT broker
func (a *Adapter) SendData(deviceID string, data map[string]interface{}) error {
	// 为每个设备创建独立连接（首次发送时创建）
	client, ok := a.clients[deviceID]
	if !ok || !client.IsConnected() {
		opts := mqttlib.NewClientOptions().
			AddBroker(a.broker).
			SetClientID(deviceID). // 用设备 ID 作为 clientID
			SetUsername(a.username).
			SetPassword(a.password).
			SetCleanSession(a.cleanSession).
			SetAutoReconnect(false).
			SetConnectionLostHandler(func(c mqttlib.Client, err error) {
				a.logger.Error().Str("device_id", deviceID).Err(err).Msg("MQTT connection lost")
			})

		client = mqttlib.NewClient(opts)
		token := client.Connect()
		token.Wait()
		if token.Error() != nil {
			return fmt.Errorf("device %s failed to connect: %w", deviceID, token.Error())
		}
		a.clients[deviceID] = client
		a.logger.Info().Str("device_id", deviceID).Msg("MQTT connected")
	}

	// Format topic
	topic := a.formatTopic(deviceID, data)

	// Marshal data to JSON
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Publish
	token := client.Publish(topic, a.qos, false, payload)
	token.Wait()

	if token.Error() != nil {
		return fmt.Errorf("failed to publish to %s: %w", topic, token.Error())
	}

	a.logger.Debug().
		Str("device_id", deviceID).
		Str("topic", topic).
		Int("bytes", len(payload)).
		Msg("Data published")

	return nil
}

// formatTopic formats the MQTT topic
func (a *Adapter) formatTopic(deviceID string, data map[string]interface{}) string {
	domainID := "default"
	region := "default"

	if v, ok := data["domain_id"]; ok {
		if s, ok := v.(string); ok {
			domainID = s
		}
	}
	if v, ok := data["region"]; ok {
		if s, ok := v.(string); ok {
			region = s
		}
	}

	topic := a.topicFormat
	if topic == "" {
		topic = "devices/{domain_id}/{region}/{device_id}/telemetry"
	}

	// Replace placeholders
	topic = replacePlaceholder(topic, "{domain_id}", domainID)
	topic = replacePlaceholder(topic, "{region}", region)
	topic = replacePlaceholder(topic, "{device_id}", deviceID)

	return topic
}

// replacePlaceholder replaces a placeholder in a string
func replacePlaceholder(s, placeholder, value string) string {
	result := ""
	i := 0
	for i < len(s) {
		if i+len(placeholder) <= len(s) && s[i:i+len(placeholder)] == placeholder {
			result += value
			i += len(placeholder)
		} else {
			result += string(s[i])
			i++
		}
	}
	return result
}
