package tcp

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
			name:  "data frame with payload",
			frame: Frame{MsgType: FrameTypeData, Payload: []byte("hello")},
		},
		{
			name:  "command frame",
			frame: Frame{MsgType: FrameTypeCommand, Payload: []byte{0x01, 0x02, 0x03}},
		},
		{
			name:  "ping frame - empty payload",
			frame: Frame{MsgType: FrameTypePing, Payload: nil},
		},
		{
			name:  "pong frame - empty payload",
			frame: Frame{MsgType: FrameTypePong, Payload: []byte{}},
		},
		{
			name:  "auth frame",
			frame: Frame{MsgType: FrameTypeAuth, Payload: []byte("token:abc123")},
		},
		{
			name:  "auth ok frame",
			frame: Frame{MsgType: FrameTypeAuthOK, Payload: nil},
		},
		{
			name:  "large payload",
			frame: Frame{MsgType: FrameTypeData, Payload: bytes.Repeat([]byte("X"), 1024)},
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

			if got.MsgType != tt.frame.MsgType {
				t.Errorf("MsgType: got 0x%04X, want 0x%04X", got.MsgType, tt.frame.MsgType)
			}
			if !bytes.Equal(got.Payload, tt.frame.Payload) {
				t.Errorf("Payload: got %d bytes, want %d bytes", len(got.Payload), len(tt.frame.Payload))
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

func TestReadFrame_TooLarge(t *testing.T) {
	// 手动构造一个超大长度的帧头
	var buf bytes.Buffer
	// length = 2MB (超过 1MB 限制)
	buf.Write([]byte{0x00, 0x20, 0x00, 0x00})
	buf.Write([]byte{0x00, 0x01})

	_, err := ReadFrame(&buf)
	if err == nil {
		t.Errorf("expected error for frame too large")
	}
}

func TestFrameConstants(t *testing.T) {
	// 验证常量值
	if FrameTypeData != 0x0001 {
		t.Errorf("FrameTypeData: got 0x%04X, want 0x0001", FrameTypeData)
	}
	if FrameTypeCommand != 0x0002 {
		t.Errorf("FrameTypeCommand: got 0x%04X, want 0x0002", FrameTypeCommand)
	}
	if FrameTypePing != 0x0010 {
		t.Errorf("FrameTypePing: got 0x%04X, want 0x0010", FrameTypePing)
	}
}
