package tcp

import (
	"fmt"
	"net"
	"time"

	"github.com/datacenter/internal/gateway"
	"github.com/datacenter/internal/message"
)

type Client struct {
	conn      net.Conn
	publisher gateway.Publisher
	onData    DeviceStatusCallback
	onAuth    func(deviceID string) // 认证成功回调
	deviceID  string
	domainID  string
	connected bool
	lastSeen  time.Time
}

func NewClient(conn net.Conn, publisher gateway.Publisher) *Client {
	return &Client{
		conn:      conn,
		publisher: publisher,
		connected: false,
	}
}

func (c *Client) ReadLoop() error {
	fmt.Printf("[TCP-GW] 🔄 ReadLoop 启动，等待数据...\n")
	for {
		frame, err := ReadFrame(c.conn)
		if err != nil {
			fmt.Printf("[TCP-GW] ❌ ReadFrame 错误: %v\n", err)
			return err
		}

		switch frame.MsgType {
		case FrameTypeAuth:
			if err := c.handleAuth(frame); err != nil {
				return err
			}
		case FrameTypeData:
			if err := c.handleData(frame); err != nil {
				fmt.Printf("[TCP-GW] ❌ data error: %v\n", err)
			}
		case FrameTypePing:
			c.handlePing()
		default:
			fmt.Printf("[TCP-GW] ⚠️ 未知帧类型: %d\n", frame.MsgType)
		}
	}
}

func (c *Client) handleAuth(frame *Frame) error {
	// Simple auth: payload = device_id|domain_id
	data := string(frame.Payload)
	for i, ch := range data {
		if ch == '|' {
			c.deviceID = data[:i]
			c.domainID = data[i+1:]
			break
		}
	}
	if c.deviceID == "" {
		c.deviceID = "unknown"
	}
	if c.domainID == "" {
		c.domainID = "default"
	}
	c.connected = true
	c.lastSeen = time.Now()

	resp := &Frame{MsgType: FrameTypeAuthOK, Payload: []byte("ok")}
	if err := WriteFrame(c.conn, resp); err != nil {
		return err
	}

	// 通知网关设备已认证
	if c.onAuth != nil {
		c.onAuth(c.deviceID)
	}

	fmt.Println("[TCP] client authenticated:", c.deviceID)
	return nil
}

func (c *Client) handleData(frame *Frame) error {
	c.lastSeen = time.Now()
	if c.onData != nil {
		c.onData(c.deviceID)
	}

	fmt.Printf("[TCP-GW] 📨 收到数据: device=%s domain=%s payload_len=%d\n",
		c.deviceID, c.domainID, len(frame.Payload))

	env := message.NewDeviceEnvelope(c.deviceID, c.domainID, "tcp_device", message.DataType)
	env.Metadata["protocol"] = "tcp"
	env.AddUnit("data", frame.Payload)

	fmt.Printf("[TCP-GW] 📦 封装 Envelope: device=%s units=%d\n", env.DeviceID, len(env.Units))

	if err := c.publisher.PublishEnvelope(env); err != nil {
		fmt.Printf("[TCP-GW] ❌ PublishEnvelope 失败: %v\n", err)
		return err
	}

	fmt.Printf("[TCP-GW] ✅ Envelope 已发布: device=%s\n", c.deviceID)
	return nil
}

func (c *Client) handlePing() {
	c.lastSeen = time.Now()
	resp := &Frame{MsgType: FrameTypePong}
	WriteFrame(c.conn, resp)
}

func (c *Client) SendCommand(payload []byte) error {
	frame := &Frame{MsgType: FrameTypeCommand, Payload: payload}
	return WriteFrame(c.conn, frame)
}
