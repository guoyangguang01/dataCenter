package tcp

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Frame 自定义协议帧格式
// [4 bytes length][2 bytes msg_type][payload]
type Frame struct {
	MsgType uint16
	Payload []byte
}

func ReadFrame(r io.Reader) (*Frame, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lenBuf[:])
	if length > 1024*1024 { // 1MB max
		return nil, fmt.Errorf("frame too large: %d bytes", length)
	}
	var typeBuf [2]byte
	if _, err := io.ReadFull(r, typeBuf[:]); err != nil {
		return nil, err
	}
	payload := make([]byte, length-2)
	if length > 2 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return nil, err
		}
	}
	return &Frame{MsgType: binary.BigEndian.Uint16(typeBuf[:]), Payload: payload}, nil
}

func WriteFrame(w io.Writer, frame *Frame) error {
	length := uint32(2 + len(frame.Payload))
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], length)
	if _, err := w.Write(lenBuf[:]); err != nil {
		return err
	}
	var typeBuf [2]byte
	binary.BigEndian.PutUint16(typeBuf[:], frame.MsgType)
	if _, err := w.Write(typeBuf[:]); err != nil {
		return err
	}
	if len(frame.Payload) > 0 {
		if _, err := w.Write(frame.Payload); err != nil {
			return err
		}
	}
	return nil
}

const (
	FrameTypeData    uint16 = 0x0001
	FrameTypeCommand uint16 = 0x0002
	FrameTypePing    uint16 = 0x0010
	FrameTypePong    uint16 = 0x0011
	FrameTypeAuth    uint16 = 0x0020
	FrameTypeAuthOK  uint16 = 0x0021
)
