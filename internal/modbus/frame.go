package modbus

import (
	"encoding/binary"
	"io"
)

// Modbus TCP 帧格式 (MBAP Header)
// [2 bytes transaction ID][2 bytes protocol ID][2 bytes length][1 byte unit ID][1 byte function code][payload]
type Frame struct {
	TransactionID uint16
	UnitID        byte
	FunctionCode  byte
	Payload       []byte
}

func ReadFrame(r io.Reader) (*Frame, error) {
	var header [7]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint16(header[4:6])
	payload := make([]byte, length-1) // minus unit ID
	if length > 1 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return nil, err
		}
	}
	return &Frame{
		TransactionID: binary.BigEndian.Uint16(header[:2]),
		UnitID:        header[6],
		FunctionCode:  payload[0],
		Payload:       payload[1:],
	}, nil
}

func WriteFrame(w io.Writer, frame *Frame) error {
	dataPayload := append([]byte{frame.FunctionCode}, frame.Payload...)
	length := uint16(1 + len(dataPayload))
	var header [7]byte
	binary.BigEndian.PutUint16(header[:2], frame.TransactionID)
	binary.BigEndian.PutUint16(header[2:4], 0) // protocol ID
	binary.BigEndian.PutUint16(header[4:6], length)
	header[6] = frame.UnitID
	if _, err := w.Write(header[:]); err != nil {
		return err
	}
	_, err := w.Write(dataPayload)
	return err
}

const (
	FuncReadCoils          byte = 0x01
	FuncReadDiscreteInputs byte = 0x02
	FuncReadHoldingRegs    byte = 0x03
	FuncReadInputRegs      byte = 0x04
	FuncWriteSingleReg     byte = 0x06
	FuncWriteMultipleRegs  byte = 0x10
)
