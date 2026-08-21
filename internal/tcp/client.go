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
	for {
		frame, err := ReadFrame(c.conn)
		if err != nil {
			return err
		}

		switch frame.MsgType {
		case FrameTypeAuth:
			if err := c.handleAuth(frame); err != nil {
				return err
			}
		case FrameTypeData:
			if err := c.handleData(frame); err != nil {
				fmt.Println("[TCP] data error:", err)
			}
		case FrameTypePing:
			c.handlePing()
		default:
			fmt.Println("[TCP] unknown frame type:", frame.MsgType)
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
	fmt.Println("[TCP] client authenticated:", c.deviceID)
	return nil
}

func (c *Client) handleData(frame *Frame) error {
	c.lastSeen = time.Now()
	if c.onData != nil {
		c.onData(c.deviceID)
	}
	env := message.NewDeviceEnvelope(c.deviceID, c.domainID, "tcp_device", message.DataType)
	env.Metadata["protocol"] = "tcp"
	env.AddUnit("data", frame.Payload)
	return c.publisher.PublishEnvelope(env)
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
