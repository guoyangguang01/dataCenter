package modbus

import (
	"bytes"
	"testing"
)

func TestWriteReadFrame_RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		frame Frame
	}{
		{
			name: "read holding registers",
			frame: Frame{
				TransactionID: 1,
				UnitID:        0x01,
				FunctionCode:  FuncReadHoldingRegs,
				Payload:       []byte{0x00, 0x00, 0x00, 0x0A}, // start=0, count=10
			},
		},
		{
			name: "read coils",
			frame: Frame{
				TransactionID: 42,
				UnitID:        0x02,
				FunctionCode:  FuncReadCoils,
				Payload:       []byte{0x00, 0x10, 0x00, 0x08}, // start=16, count=8
			},
		},
		{
			name: "write single register",
			frame: Frame{
				TransactionID: 100,
				UnitID:        0x01,
				FunctionCode:  FuncWriteSingleReg,
				Payload:       []byte{0x00, 0x01, 0x00, 0xFF}, // addr=1, value=255
			},
		},
		{
			name: "write multiple registers",
			frame: Frame{
				TransactionID: 200,
				UnitID:        0x03,
				FunctionCode:  FuncWriteMultipleRegs,
				Payload:       []byte{0x00, 0x00, 0x00, 0x02, 0x04, 0x00, 0x0A, 0x00, 0x14},
			},
		},
		{
			name: "empty payload",
			frame: Frame{
				TransactionID: 0,
				UnitID:        0x01,
				FunctionCode:  FuncReadInputRegs,
				Payload:       nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteFrame(&buf, &tt.frame); err != nil {
				t.Fatalf("WriteFrame error: %v", err)
			}

			got, err := ReadFrame(&buf)
			if err != nil {
				t.Fatalf("ReadFrame error: %v", err)
			}

			if got.TransactionID != tt.frame.TransactionID {
				t.Errorf("TransactionID: got %d, want %d", got.TransactionID, tt.frame.TransactionID)
			}
			if got.UnitID != tt.frame.UnitID {
				t.Errorf("UnitID: got %d, want %d", got.UnitID, tt.frame.UnitID)
			}
			if got.FunctionCode != tt.frame.FunctionCode {
				t.Errorf("FunctionCode: got 0x%02X, want 0x%02X", got.FunctionCode, tt.frame.FunctionCode)
			}
			if !bytes.Equal(got.Payload, tt.frame.Payload) {
				t.Errorf("Payload: got %v, want %v", got.Payload, tt.frame.Payload)
			}
		})
	}
}

func TestReadFrame_EmptyReader(t *testing.T) {
	var buf bytes.Buffer
	_, err := ReadFrame(&buf)
	if err == nil {
		t.Errorf("expected error for empty reader")
	}
}

func TestModbusConstants(t *testing.T) {
	if FuncReadCoils != 0x01 {
		t.Errorf("FuncReadCoils: got 0x%02X, want 0x01", FuncReadCoils)
	}
	if FuncReadHoldingRegs != 0x03 {
		t.Errorf("FuncReadHoldingRegs: got 0x%02X, want 0x03", FuncReadHoldingRegs)
	}
	if FuncWriteSingleReg != 0x06 {
		t.Errorf("FuncWriteSingleReg: got 0x%02X, want 0x06", FuncWriteSingleReg)
	}
}
