package tcp

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

// Frame types matching the TCP gateway protocol
const (
	FrameTypeData   = 0x0001
	FrameTypeCommand = 0x0002
	FrameTypePing    = 0x0010
	FrameTypePong    = 0x0011
	FrameTypeAuth    = 0x0020
	FrameTypeAuthOK  = 0x0021
)

// Adapter implements the TCP protocol adapter
type Adapter struct {
	host     string
	port     int
	mode     string // client or server
	logger   zerolog.Logger

	// Client mode
	conn     net.Conn

	// Server mode
	listener net.Listener
	clients  map[string]net.Conn
	mu       sync.RWMutex
}

// NewAdapter creates a new TCP adapter
func NewAdapter(host string, port int, mode string, logger zerolog.Logger) *Adapter {
	return &Adapter{
		host:   host,
		port:   port,
		mode:   mode,
		logger: logger,
		clients: make(map[string]net.Conn),
	}
}

// Start starts the TCP adapter
func (a *Adapter) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", a.host, a.port)

	switch a.mode {
	case "client":
		return a.startClient(ctx, addr)
	case "server":
		return a.startServer(ctx, addr)
	default:
		return fmt.Errorf("unsupported TCP mode: %s", a.mode)
	}
}

// startClient starts TCP client mode
func (a *Adapter) startClient(ctx context.Context, addr string) error {
	a.logger.Info().
		Str("address", addr).
		Msg("Connecting to TCP server")

	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to TCP server: %w", err)
	}

	a.conn = conn
	a.logger.Info().
		Str("address", addr).
		Msg("Connected to TCP server")

	return nil
}

// startServer starts TCP server mode
func (a *Adapter) startServer(ctx context.Context, addr string) error {
	a.logger.Info().
		Str("address", addr).
		Msg("Starting TCP server")

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start TCP server: %w", err)
	}

	a.listener = listener

	// Accept connections in background
	go a.acceptLoop(ctx)

	a.logger.Info().
		Str("address", addr).
		Msg("TCP server started")

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

// handleConnection handles an incoming connection
func (a *Adapter) handleConnection(ctx context.Context, conn net.Conn) {
	a.logger.Info().
		Str("remote", conn.RemoteAddr().String()).
		Msg("New connection")

	// Wait for auth frame
	frame, err := ReadFrame(conn)
	if err != nil {
		a.logger.Error().Err(err).Msg("Failed to read auth frame")
		conn.Close()
		return
	}

	if frame.MsgType != FrameTypeAuth {
		a.logger.Error().
			Uint16("type", frame.MsgType).
			Msg("Expected auth frame")
		conn.Close()
		return
	}

	// Parse auth payload: device_id|domain_id
	authData := string(frame.Payload)
	deviceID := authData
	// Simple parsing - in production, use proper format
	if len(authData) > 0 {
		a.mu.Lock()
		a.clients[deviceID] = conn
		a.mu.Unlock()
	}

	// Send auth OK
	authOK := EncodeFrame(FrameTypeAuthOK, []byte("OK"))
	_, err = conn.Write(authOK)
	if err != nil {
		a.logger.Error().Err(err).Msg("Failed to send auth OK")
		conn.Close()
		return
	}

	a.logger.Info().
		Str("device_id", deviceID).
		Msg("Device authenticated")

	// Handle pings in background
	go a.handleClientPings(ctx, conn, deviceID)
}

// handleClientPings handles ping frames from a client
func (a *Adapter) handleClientPings(ctx context.Context, conn net.Conn, deviceID string) {
	defer func() {
		a.mu.Lock()
		delete(a.clients, deviceID)
		a.mu.Unlock()
		conn.Close()
		a.logger.Info().
			Str("device_id", deviceID).
			Msg("Device disconnected")
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		frame, err := ReadFrame(conn)
		if err != nil {
			if err != io.EOF {
				a.logger.Error().Err(err).
					Str("device_id", deviceID).
					Msg("Failed to read frame")
			}
			return
		}

		switch frame.MsgType {
		case FrameTypePing:
			pong := EncodeFrame(FrameTypePong, []byte{})
			_, err = conn.Write(pong)
			if err != nil {
				a.logger.Error().Err(err).
					Str("device_id", deviceID).
					Msg("Failed to send pong")
				return
			}
		default:
			a.logger.Warn().
				Uint16("type", frame.MsgType).
				Str("device_id", deviceID).
				Msg("Unexpected frame type from client")
		}
	}
}

// Stop stops the TCP adapter
func (a *Adapter) Stop() error {
	if a.conn != nil {
		a.conn.Close()
	}
	if a.listener != nil {
		a.listener.Close()
	}
	a.logger.Info().Msg("TCP adapter stopped")
	return nil
}

// SendData sends data via TCP
func (a *Adapter) SendData(deviceID string, data map[string]interface{}) error {
	switch a.mode {
	case "client":
		return a.sendDataClient(deviceID, data)
	case "server":
		return a.sendDataServer(deviceID, data)
	default:
		return fmt.Errorf("unsupported mode: %s", a.mode)
	}
}

// sendDataClient sends data in client mode
func (a *Adapter) sendDataClient(deviceID string, data map[string]interface{}) error {
	if a.conn == nil {
		a.logger.Error().Str("device_id", deviceID).Msg("[TCP-Sim] ❌ 未连接到服务器")
		return fmt.Errorf("not connected to server")
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	a.logger.Info().
		Str("device_id", deviceID).
		Int("payload_bytes", len(payload)).
		Msg("[TCP-Sim] 📤 发送数据...")

	frame := EncodeFrame(FrameTypeData, payload)
	_, err = a.conn.Write(frame)
	if err != nil {
		a.logger.Error().
			Str("device_id", deviceID).
			Err(err).
			Msg("[TCP-Sim] ❌ 发送失败")
		return fmt.Errorf("failed to send data: %w", err)
	}

	a.logger.Info().
		Str("device_id", deviceID).
		Int("payload_bytes", len(payload)).
		Msg("[TCP-Sim] ✅ 发送成功")

	return nil
}

// sendDataServer sends data in server mode (to a specific client)
func (a *Adapter) sendDataServer(deviceID string, data map[string]interface{}) error {
	a.mu.RLock()
	conn, ok := a.clients[deviceID]
	a.mu.RUnlock()

	if !ok {
		return fmt.Errorf("device %s not connected", deviceID)
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	frame := EncodeFrame(FrameTypeData, payload)
	_, err = conn.Write(frame)
	if err != nil {
		return fmt.Errorf("failed to send data: %w", err)
	}

	a.logger.Debug().
		Str("device_id", deviceID).
		Int("bytes", len(payload)).
		Msg("Data sent to client")

	return nil
}

// Frame represents a TCP protocol frame
type Frame struct {
	Length  uint32
	MsgType uint16
	Payload []byte
}

// EncodeFrame encodes a frame to bytes
func EncodeFrame(msgType uint16, payload []byte) []byte {
	length := uint32(2 + len(payload)) // 2 bytes for msgType + payload
	frame := make([]byte, 4+2+len(payload))

	binary.BigEndian.PutUint32(frame[0:4], length)
	binary.BigEndian.PutUint16(frame[4:6], msgType)
	copy(frame[6:], payload)

	return frame
}

// ReadFrame reads a frame from a connection
func ReadFrame(conn net.Conn) (*Frame, error) {
	// Read length (4 bytes)
	lengthBuf := make([]byte, 4)
	_, err := io.ReadFull(conn, lengthBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to read frame length: %w", err)
	}

	length := binary.BigEndian.Uint32(lengthBuf)
	if length > 1024*1024 { // Max 1MB
		return nil, fmt.Errorf("frame too large: %d bytes", length)
	}

	// Read msgType (2 bytes)
	msgTypeBuf := make([]byte, 2)
	_, err = io.ReadFull(conn, msgTypeBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to read frame type: %w", err)
	}

	msgType := binary.BigEndian.Uint16(msgTypeBuf)

	// Read payload
	payloadLen := length - 2
	payload := make([]byte, payloadLen)
	if payloadLen > 0 {
		_, err = io.ReadFull(conn, payload)
		if err != nil {
			return nil, fmt.Errorf("failed to read frame payload: %w", err)
		}
	}

	return &Frame{
		Length:  length,
		MsgType: msgType,
		Payload: payload,
	}, nil
}
