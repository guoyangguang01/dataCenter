package modbus

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/datacenter/internal/gateway"
	"github.com/datacenter/internal/message"
)

type Client struct {
	conn       net.Conn
	publisher  gateway.Publisher
	onData     DeviceStatusCallback
	config     Config
	unitID     byte
	writeQueue chan []byte // 下行写命令队列
}

func NewClient(conn net.Conn, publisher gateway.Publisher, config Config) *Client {
	unitID := byte(1)
	if len(config.SlaveIDs) > 0 {
		unitID = byte(config.SlaveIDs[0])
	}
	return &Client{
		conn:       conn,
		publisher:  publisher,
		config:     config,
		unitID:     unitID,
		writeQueue: make(chan []byte, 16), // 缓冲 16 条写命令
	}
}

// SendCommand 将下行命令入队到写队列
func (c *Client) SendCommand(payload []byte) error {
	select {
	case c.writeQueue <- payload:
		fmt.Printf("[Modbus-GW] 📤 命令已入队: unitID=%d queueLen=%d\n", c.unitID, len(c.writeQueue))
		return nil
	default:
		return fmt.Errorf("write queue full for unitID %d", c.unitID)
	}
}

func (c *Client) ReadLoop() {
	fmt.Printf("[Modbus-GW] 🔄 ReadLoop 启动，等待数据...\n")
	for {
		// 非阻塞检查写队列
		select {
		case payload := <-c.writeQueue:
			c.handleWriteQueue(payload)
			continue
		default:
		}

		// 设置短暂的读超时，以便定期检查写队列
		c.conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		frame, err := ReadFrame(c.conn)
		if err != nil {
			// 超时不是错误，继续检查写队列
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			fmt.Printf("[Modbus-GW] ❌ read error: %v\n", err)
			return
		}
		fmt.Printf("[Modbus-GW] 📨 收到帧: funcCode=%d unitID=%d payloadLen=%d\n",
			frame.FunctionCode, frame.UnitID, len(frame.Payload))
		c.handleFrame(frame)
	}
}

// handleWriteQueue 处理写队列中的下行命令
func (c *Client) handleWriteQueue(payload []byte) {
	// payload 格式: [registerAddr_high, registerAddr_low, value_high, value_low]
	// 构造 Write Single Register (0x06) 响应
	if len(payload) < 4 {
		fmt.Printf("[Modbus-GW] ❌ 写命令 payload 太短: %d bytes\n", len(payload))
		return
	}

	resp := &Frame{
		UnitID:       c.unitID,
		FunctionCode: FuncWriteSingleReg,
		Payload:      payload, // [addr_h, addr_l, value_h, value_l]
	}
	if err := WriteFrame(c.conn, resp); err != nil {
		fmt.Printf("[Modbus-GW] ❌ 写响应发送失败: %v\n", err)
		return
	}
	fmt.Printf("[Modbus-GW] ✅ 写命令已发送: unitID=%d register=%02x%02x\n",
		c.unitID, payload[0], payload[1])
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
