package message

import "time"

// MessageType 消息类型
type MessageType int32

const (
	DataType    MessageType = 0 // 设备上报数据
	CommandType MessageType = 1 // 下发指令
	EventType   MessageType = 2 // 设备事件
	ACKType     MessageType = 3 // 确认
)

// QoSLevel QoS 等级
type QoSLevel int32

const (
	ATMOST_ONCE   QoSLevel = 0
	ATLEAST_ONCE  QoSLevel = 1
	EXACTLY_ONCE  QoSLevel = 2
)

// DeviceEnvelope 顶层信封
type DeviceEnvelope struct {
	DeviceID string            `json:"device_id"`
	DomainID string            `json:"domain_id"`
	ModelID  string            `json:"model_id"`
	Type     MessageType       `json:"type"`
	Timestamp int64            `json:"timestamp"`
	Units    []MessageUnit     `json:"units"`
	QoS      QoSLevel          `json:"qos"`
	Metadata map[string]string `json:"metadata"`
}

// MessageUnit 单个数据单元
type MessageUnit struct {
	Topic     string            `json:"topic"`
	Payload   []byte            `json:"payload"`
	Timestamp int64             `json:"timestamp"`
	Metadata  map[string]string `json:"metadata"`
}

// NewDeviceEnvelope 创建设备消息信封
func NewDeviceEnvelope(deviceID, domainID, modelID string, msgType MessageType) *DeviceEnvelope {
	return &DeviceEnvelope{
		DeviceID:  deviceID,
		DomainID:  domainID,
		ModelID:   modelID,
		Type:      msgType,
		Timestamp: time.Now().UnixMilli(),
		Units:     make([]MessageUnit, 0),
		Metadata:  make(map[string]string),
	}
}

// AddUnit 添加数据单元
func (e *DeviceEnvelope) AddUnit(topic string, payload []byte) *DeviceEnvelope {
	e.Units = append(e.Units, MessageUnit{
		Topic:     topic,
		Payload:   payload,
		Timestamp: time.Now().UnixMilli(),
		Metadata:  make(map[string]string),
	})
	return e
}
