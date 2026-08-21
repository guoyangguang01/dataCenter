package modbus

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Modbus function codes
const (
	FuncReadCoils          = 0x01
	FuncReadDiscreteInputs = 0x02
	FuncReadHoldingRegs    = 0x03
	FuncReadInputRegs      = 0x04
	FuncWriteSingleReg     = 0x06
	FuncWriteMultipleRegs  = 0x10
)

// MBAP Header size
const MBAPHeaderSize = 7

// Adapter implements the Modbus protocol adapter
type Adapter struct {
	host     string
	port     int
	mode     string // master or slave
	slaveIDs []byte
	logger   zerolog.Logger

	// Slave mode
	listener   net.Listener
	registers  map[byte]map[uint16][]byte // slaveID -> address -> data
	mu         sync.RWMutex

	// Master mode
	conn       net.Conn
}

// NewAdapter creates a new Modbus adapter
func NewAdapter(host string, port int, mode string, slaveIDs []byte, logger zerolog.Logger) *Adapter {
	if len(slaveIDs) == 0 {
		slaveIDs = []byte{1}
	}
	return &Adapter{
		host:       host,
		port:       port,
		mode:       mode,
		slaveIDs:   slaveIDs,
		logger:     logger,
		registers:  make(map[byte]map[uint16][]byte),
	}
}

// Start starts the Modbus adapter
func (a *Adapter) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", a.host, a.port)

	switch a.mode {
	case "slave":
		return a.startSlave(ctx, addr)
	case "master":
		return a.startMaster(ctx, addr)
	default:
		return fmt.Errorf("unsupported Modbus mode: %s", a.mode)
	}
}

// startSlave starts Modbus slave mode
func (a *Adapter) startSlave(ctx context.Context, addr string) error {
	a.logger.Info().
		Str("address", addr).
		Msg("Starting Modbus slave")

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start Modbus slave: %w", err)
	}

	a.listener = listener

	// Initialize registers for each slave
	for _, slaveID := range a.slaveIDs {
		a.registers[slaveID] = make(map[uint16][]byte)
	}

	// Accept connections in background
	go a.acceptLoop(ctx)

	a.logger.Info().
		Str("address", addr).
		Int("slaves", len(a.slaveIDs)).
		Msg("Modbus slave started")

	return nil
}

// startMaster starts Modbus master mode
func (a *Adapter) startMaster(ctx context.Context, addr string) error {
	a.logger.Info().
		Str("address", addr).
		Msg("Connecting to Modbus slave")

	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to Modbus slave: %w", err)
	}

	a.conn = conn
	a.logger.Info().
		Str("address", addr).
		Msg("Connected to Modbus slave")

	return nil
}

// acceptLoop accepts incoming connections
func (a *Adapter) acceptLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := a.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				a.logger.Error().Err(err).Msg("Failed to accept connection")
				continue
			}
		}

		go a.handleConnection(ctx, conn)
	}
}

// handleConnection handles an incoming Modbus connection
func (a *Adapter) handleConnection(ctx context.Context, conn net.Conn) {
	a.logger.Info().
		Str("remote", conn.RemoteAddr().String()).
		Msg("New Modbus connection")

	defer conn.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Read MBAP header
		header := make([]byte, MBAPHeaderSize)
		_, err := io.ReadFull(conn, header)
		if err != nil {
			if err != io.EOF {
				a.logger.Error().Err(err).Msg("Failed to read MBAP header")
			}
			return
		}

		// Parse MBAP header
		txID := binary.BigEndian.Uint16(header[0:2])
		protocolID := binary.BigEndian.Uint16(header[2:4])
		length := binary.BigEndian.Uint16(header[4:6])
		unitID := header[6]

		if protocolID != 0 {
			a.logger.Warn().
				Uint16("protocol_id", protocolID).
				Msg("Invalid Modbus protocol ID")
			continue
		}

		// Read PDU
		pduLen := int(length) - 1
		if pduLen <= 0 {
			continue
		}
		pdu := make([]byte, pduLen)
		_, err = io.ReadFull(conn, pdu)
		if err != nil {
			a.logger.Error().Err(err).Msg("Failed to read PDU")
			return
		}

		// Handle request
		response := a.handleRequest(txID, unitID, pdu)
		if response != nil {
			_, err = conn.Write(response)
			if err != nil {
				a.logger.Error().Err(err).Msg("Failed to send response")
				return
			}
		}
	}
}

// handleRequest handles a Modbus request
func (a *Adapter) handleRequest(txID uint16, unitID byte, pdu []byte) []byte {
	if len(pdu) < 1 {
		return nil
	}

	funcCode := pdu[0]

	switch funcCode {
	case FuncReadHoldingRegs:
		return a.handleReadHoldingRegs(txID, unitID, pdu)
	case FuncReadInputRegs:
		return a.handleReadInputRegs(txID, unitID, pdu)
	case FuncWriteSingleReg:
		return a.handleWriteSingleReg(txID, unitID, pdu)
	default:
		a.logger.Warn().
			Uint8("func_code", funcCode).
			Msg("Unsupported function code")
		return nil
	}
}

// handleReadHoldingRegs handles read holding registers request
func (a *Adapter) handleReadHoldingRegs(txID uint16, unitID byte, pdu []byte) []byte {
	if len(pdu) < 5 {
		return nil
	}

	startAddr := binary.BigEndian.Uint16(pdu[1:3])
	quantity := binary.BigEndian.Uint16(pdu[3:5])

	a.logger.Debug().
		Uint8("unit_id", unitID).
		Uint16("start_addr", startAddr).
		Uint16("quantity", quantity).
		Msg("Read holding registers")

	// Get or create register data
	a.mu.RLock()
	slaveRegs, ok := a.registers[unitID]
	a.mu.RUnlock()

	if !ok {
		a.mu.Lock()
		a.registers[unitID] = make(map[uint16][]byte)
		slaveRegs = a.registers[unitID]
		a.mu.Unlock()
	}

	// Generate response data
	byteCount := quantity * 2
	data := make([]byte, byteCount)
	for i := uint16(0); i < quantity; i++ {
		addr := startAddr + i
		a.mu.RLock()
		regData, exists := slaveRegs[addr]
		a.mu.RUnlock()

		if exists && len(regData) >= 2 {
			data[i*2] = regData[0]
			data[i*2+1] = regData[1]
		} else {
			// Generate default data
			val := uint16(1000 + addr)
			binary.BigEndian.PutUint16(data[i*2:], val)
		}
	}

	// Build response PDU
	responsePDU := make([]byte, 2+len(data))
	responsePDU[0] = FuncReadHoldingRegs
	responsePDU[1] = byte(len(data))
	copy(responsePDU[2:], data)

	// Build MBAP header
	response := a.buildMBAPResponse(txID, unitID, responsePDU)

	return response
}

// handleReadInputRegs handles read input registers request
func (a *Adapter) handleReadInputRegs(txID uint16, unitID byte, pdu []byte) []byte {
	// Same as read holding registers
	return a.handleReadHoldingRegs(txID, unitID, pdu)
}

// handleWriteSingleReg handles write single register request
func (a *Adapter) handleWriteSingleReg(txID uint16, unitID byte, pdu []byte) []byte {
	if len(pdu) < 5 {
		return nil
	}

	addr := binary.BigEndian.Uint16(pdu[1:3])
	value := binary.BigEndian.Uint16(pdu[3:5])

	a.logger.Debug().
		Uint8("unit_id", unitID).
		Uint16("addr", addr).
		Uint16("value", value).
		Msg("Write single register")

	// Store register value
	a.mu.Lock()
	if _, ok := a.registers[unitID]; !ok {
		a.registers[unitID] = make(map[uint16][]byte)
	}
	regData := make([]byte, 2)
	binary.BigEndian.PutUint16(regData, value)
	a.registers[unitID][addr] = regData
	a.mu.Unlock()

	// Echo back as response
	responsePDU := make([]byte, 5)
	responsePDU[0] = FuncWriteSingleReg
	binary.BigEndian.PutUint16(responsePDU[1:3], addr)
	binary.BigEndian.PutUint16(responsePDU[3:5], value)

	return a.buildMBAPResponse(txID, unitID, responsePDU)
}

// buildMBAPResponse builds a Modbus TCP response
func (a *Adapter) buildMBAPResponse(txID uint16, unitID byte, pdu []byte) []byte {
	length := uint16(1 + len(pdu)) // unitID + PDU
	response := make([]byte, MBAPHeaderSize+len(pdu))

	binary.BigEndian.PutUint16(response[0:2], txID)
	binary.BigEndian.PutUint16(response[2:4], 0) // Protocol ID
	binary.BigEndian.PutUint16(response[4:6], length)
	response[6] = unitID
	copy(response[7:], pdu)

	return response
}

// Stop stops the Modbus adapter
func (a *Adapter) Stop() error {
	if a.conn != nil {
		a.conn.Close()
	}
	if a.listener != nil {
		a.listener.Close()
	}
	a.logger.Info().Msg("Modbus adapter stopped")
	return nil
}

// SendData sends data via Modbus
func (a *Adapter) SendData(deviceID string, data map[string]interface{}) error {
	switch a.mode {
	case "slave":
		return a.sendDataSlave(deviceID, data)
	case "master":
		return a.sendDataMaster(deviceID, data)
	default:
		return fmt.Errorf("unsupported mode: %s", a.mode)
	}
}

// sendDataSlave sends data in slave mode (update registers)
func (a *Adapter) sendDataSlave(deviceID string, data map[string]interface{}) error {
	// In slave mode, we update the register values
	// The master will read these values

	// Find the slave ID for this device
	slaveID := byte(1)
	for _, id := range a.slaveIDs {
		slaveID = id
		break
	}

	// Update registers with data
	payload, _ := json.Marshal(data)
	a.mu.Lock()
	if _, ok := a.registers[slaveID]; !ok {
		a.registers[slaveID] = make(map[uint16][]byte)
	}
	// Store data in holding registers starting at address 0
	for i := 0; i < len(payload) && i < 100; i += 2 {
		addr := uint16(i / 2)
		regData := make([]byte, 2)
		if i+1 < len(payload) {
			regData[0] = payload[i]
			regData[1] = payload[i+1]
		} else {
			regData[0] = payload[i]
		}
		a.registers[slaveID][addr] = regData
	}
	a.mu.Unlock()

	a.logger.Debug().
		Str("device_id", deviceID).
		Uint8("slave_id", slaveID).
		Int("bytes", len(payload)).
		Msg("Updated Modbus registers")

	return nil
}

// sendDataMaster sends data in master mode (write registers)
func (a *Adapter) sendDataMaster(deviceID string, data map[string]interface{}) error {
	if a.conn == nil {
		return fmt.Errorf("not connected to slave")
	}

	// Marshal data
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Write to slave registers
	slaveID := a.slaveIDs[0]
	startAddr := uint16(0)

	// Split payload into 2-byte register values
	for i := 0; i < len(payload); i += 2 {
		addr := startAddr + uint16(i/2)
		value := uint16(0)
		if i+1 < len(payload) {
			value = uint16(payload[i])<<8 | uint16(payload[i+1])
		} else {
			value = uint16(payload[i]) << 8
		}

		// Build write single register request
		pdu := make([]byte, 5)
		pdu[0] = FuncWriteSingleReg
		binary.BigEndian.PutUint16(pdu[1:3], addr)
		binary.BigEndian.PutUint16(pdu[3:5], value)

		// Build MBAP header
		txID := uint16(time.Now().UnixMilli() & 0xFFFF)
		length := uint16(1 + len(pdu))
		request := make([]byte, MBAPHeaderSize+len(pdu))
		binary.BigEndian.PutUint16(request[0:2], txID)
		binary.BigEndian.PutUint16(request[2:4], 0)
		binary.BigEndian.PutUint16(request[4:6], length)
		request[6] = slaveID
		copy(request[7:], pdu)

		// Send request
		_, err = a.conn.Write(request)
		if err != nil {
			return fmt.Errorf("failed to send write request: %w", err)
		}

		// Read response
		response := make([]byte, MBAPHeaderSize+len(pdu))
		_, err = io.ReadFull(a.conn, response)
		if err != nil {
			return fmt.Errorf("failed to read write response: %w", err)
		}
	}

	a.logger.Debug().
		Str("device_id", deviceID).
		Uint8("slave_id", slaveID).
		Int("bytes", len(payload)).
		Msg("Data written to Modbus slave")

	return nil
}
