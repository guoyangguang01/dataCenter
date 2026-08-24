package modbus

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/datacenter/internal/gateway"
	"github.com/datacenter/internal/message"
)

type Client struct {
	conn      net.Conn
	publisher gateway.Publisher
	onData    DeviceStatusCallback
	config    Config
	unitID    byte
}

func NewClient(conn net.Conn, publisher gateway.Publisher, config Config) *Client {
	unitID := byte(1)
	if len(config.SlaveIDs) > 0 {
		unitID = byte(config.SlaveIDs[0])
	}
	return &Client{
		conn:      conn,
		publisher: publisher,
		config:    config,
		unitID:    unitID,
	}
}

func (c *Client) ReadLoop() {
	fmt.Printf("[Modbus-GW] 🔄 ReadLoop 启动，等待数据...\n")
	for {
		frame, err := ReadFrame(c.conn)
		if err != nil {
			fmt.Printf("[Modbus-GW] ❌ read error: %v\n", err)
			return
		}
		fmt.Printf("[Modbus-GW] 📨 收到帧: funcCode=%d unitID=%d payloadLen=%d\n",
			frame.FunctionCode, frame.UnitID, len(frame.Payload))
		c.handleFrame(frame)
	}
}

func (c *Client) handleFrame(frame *Frame) {
	switch frame.FunctionCode {
	case FuncReadHoldingRegs:
		c.handleReadHoldingRegs(frame)
	case FuncReadInputRegs:
		c.handleReadInputRegs(frame)
	case FuncWriteSingleReg:
		c.handleWriteSingleReg(frame)
	default:
		fmt.Println("[Modbus] unsupported function:", frame.FunctionCode)
	}
}

func (c *Client) handleReadHoldingRegs(frame *Frame) {
	if len(frame.Payload) < 4 {
		fmt.Printf("[Modbus-GW] ❌ payload 太短: %d bytes\n", len(frame.Payload))
		return
	}
	startAddr := binary.BigEndian.Uint16(frame.Payload[:2])
	quantity := binary.BigEndian.Uint16(frame.Payload[2:4])

	fmt.Printf("[Modbus-GW] 📨 读保持寄存器: unitID=%d startAddr=%d quantity=%d\n",
		frame.UnitID, startAddr, quantity)

	data := make([]byte, quantity*2)
	for i := uint16(0); i < quantity; i++ {
		binary.BigEndian.PutUint16(data[i*2:i*2+2], uint16(1000+startAddr+i))
	}

	deviceID := fmt.Sprintf("modbus-slave-%d", frame.UnitID)
	env := message.NewDeviceEnvelope(
		deviceID,
		"default",
		"modbus_device",
		message.DataType,
	)
	env.Metadata["function_code"] = fmt.Sprintf("%d", frame.FunctionCode)
	env.Metadata["start_address"] = fmt.Sprintf("%d", startAddr)
	env.AddUnit("holding_registers", data)
	if c.onData != nil {
		c.onData(fmt.Sprintf("modbus-%d", c.unitID))
	}

	fmt.Printf("[Modbus-GW] 📦 封装 Envelope: device=%s units=%d\n", deviceID, len(env.Units))

	if err := c.publisher.PublishEnvelope(env); err != nil {
		fmt.Printf("[Modbus-GW] ❌ PublishEnvelope 失败: %v\n", err)
	} else {
		fmt.Printf("[Modbus-GW] ✅ Envelope 已发布: device=%s\n", deviceID)
	}

	resp := &Frame{
		TransactionID: frame.TransactionID,
		UnitID:        frame.UnitID,
		FunctionCode:  frame.FunctionCode,
		Payload:       data,
	}
	WriteFrame(c.conn, resp)
}

func (c *Client) handleReadInputRegs(frame *Frame) {
	c.handleReadHoldingRegs(frame)
}

func (c *Client) handleWriteSingleReg(frame *Frame) {
	resp := &Frame{
		TransactionID: frame.TransactionID,
		UnitID:        frame.UnitID,
		FunctionCode:  frame.FunctionCode,
		Payload:       frame.Payload,
	}
	WriteFrame(c.conn, resp)
	env := message.NewDeviceEnvelope(
		fmt.Sprintf("modbus-slave-%d", frame.UnitID),
		"default",
		"modbus_device",
		message.CommandType,
	)
	env.AddUnit("write_register", frame.Payload)
	c.publisher.PublishEnvelope(env)
}
