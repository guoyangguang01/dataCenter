package mqtt

import (
	"bytes"
	"testing"
)

func TestWriteReadPacket_RoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		packet  Packet
	}{
		{
			name:   "PINGREQ - empty payload",
			packet: Packet{Type: PINGREQ, Flags: 0, Payload: nil},
		},
		{
			name:   "PINGRESP - empty payload",
			packet: Packet{Type: PINGRESP, Flags: 0, Payload: nil},
		},
		{
			name:   "CONNECT - small payload",
			packet: Packet{Type: CONNECT, Flags: 0, Payload: []byte{0x00, 0x04, 'M', 'Q', 'T', 'T'}},
		},
		{
			name:   "PUBLISH - with payload",
			packet: Packet{Type: PUBLISH, Flags: 0x02, Payload: []byte("hello world")},
		},
		{
			name:   "PUBLISH - large payload",
			packet: Packet{Type: PUBLISH, Flags: 0, Payload: bytes.Repeat([]byte("A"), 256)},
		},
		{
			name:   "SUBSCRIBE",
			packet: Packet{Type: SUBSCRIBE, Flags: 0x02, Payload: []byte{0x00, 0x01, 0x00, 0x05, 't', 'e', 's', 't', '/', 'a', 0x01}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WritePacket(&buf, &tt.packet); err != nil {
				t.Fatalf("WritePacket error: %v", err)
			}

			got, err := ReadPacket(&buf)
			if err != nil {
				t.Fatalf("ReadPacket error: %v", err)
			}

			if got.Type != tt.packet.Type {
				t.Errorf("Type: got 0x%02X, want 0x%02X", got.Type, tt.packet.Type)
			}
			if got.Flags != tt.packet.Flags {
				t.Errorf("Flags: got 0x%02X, want 0x%02X", got.Flags, tt.packet.Flags)
			}
			if !bytes.Equal(got.Payload, tt.packet.Payload) {
				t.Errorf("Payload mismatch: got %d bytes, want %d bytes", len(got.Payload), len(tt.packet.Payload))
			}
		})
	}
}

func TestReadPacket_EmptyReader(t *testing.T) {
	var buf bytes.Buffer
	_, err := ReadPacket(&buf)
	if err == nil {
		t.Errorf("expected error for empty reader")
	}
}

func TestReadPacket_RemainingLength_Encoding(t *testing.T) {
	// 测试变长编码：remaining length = 200 (需要 2 字节编码)
	payload := bytes.Repeat([]byte("X"), 200)
	pkt := Packet{Type: PUBLISH, Flags: 0, Payload: payload}

	var buf bytes.Buffer
	if err := WritePacket(&buf, &pkt); err != nil {
		t.Fatalf("WritePacket error: %v", err)
	}

	got, err := ReadPacket(&buf)
	if err != nil {
		t.Fatalf("ReadPacket error: %v", err)
	}
	if len(got.Payload) != 200 {
		t.Errorf("expected 200 bytes payload, got %d", len(got.Payload))
	}
}

// --- ParseConnect 测试 ---

func TestParseConnect(t *testing.T) {
	// 构造一个简单的 CONNECT payload
	payload := []byte{
		0x00, 0x04, // protocol name length = 4
		'M', 'Q', 'T', 'T', // protocol name
		0x04,       // protocol level (MQTT 3.1.1)
		0xC0,       // connect flags (username + password)
		0x00, 0x3C, // keep alive = 60
		0x00, 0x05, // client id length = 5
		't', 'e', 's', 't', '1', // client id
	}

	pkt, err := ParseConnect(payload)
	if err != nil {
		t.Fatalf("ParseConnect error: %v", err)
	}
	if pkt.ProtocolName != "MQTT" {
		t.Errorf("expected protocol MQTT, got %s", pkt.ProtocolName)
	}
	if pkt.ProtocolLevel != 4 {
		t.Errorf("expected level 4, got %d", pkt.ProtocolLevel)
	}
	if pkt.KeepAlive != 60 {
		t.Errorf("expected keepalive 60, got %d", pkt.KeepAlive)
	}
	if pkt.ClientID != "test1" {
		t.Errorf("expected clientID test1, got %s", pkt.ClientID)
	}
}

func TestParseConnect_TooShort(t *testing.T) {
	_, err := ParseConnect([]byte{0x00, 0x01})
	if err == nil {
		t.Errorf("expected error for short payload")
	}
}

// --- ParsePublish 测试 ---

func TestParsePublish_QoS0(t *testing.T) {
	// QoS 0: no packet ID
	payload := []byte{
		0x00, 0x05, // topic length
		'h', 'e', 'l', 'l', 'o', // topic
		'd', 'a', 't', 'a', // payload data
	}

	pkt, err := ParsePublish(0, payload) // flags=0 => QoS 0
	if err != nil {
		t.Fatalf("ParsePublish error: %v", err)
	}
	if pkt.TopicName != "hello" {
		t.Errorf("expected topic hello, got %s", pkt.TopicName)
	}
	if pkt.PacketID != 0 {
		t.Errorf("expected packetID 0 for QoS 0, got %d", pkt.PacketID)
	}
	if string(pkt.Payload) != "data" {
		t.Errorf("expected payload data, got %s", string(pkt.Payload))
	}
}

func TestParsePublish_QoS1(t *testing.T) {
	// QoS 1: has packet ID
	payload := []byte{
		0x00, 0x05, // topic length
		'h', 'e', 'l', 'l', 'o', // topic
		0x00, 0x0A, // packet ID = 10
		'd', 'a', 't', 'a', // payload data
	}

	pkt, err := ParsePublish(0x02, payload) // flags=0x02 => QoS 1
	if err != nil {
		t.Fatalf("ParsePublish error: %v", err)
	}
	if pkt.PacketID != 10 {
		t.Errorf("expected packetID 10, got %d", pkt.PacketID)
	}
	if string(pkt.Payload) != "data" {
		t.Errorf("expected payload data, got %s", string(pkt.Payload))
	}
}

func TestParsePublish_TooShort(t *testing.T) {
	_, err := ParsePublish(0, []byte{0x00})
	if err == nil {
		t.Errorf("expected error for short payload")
	}
}

// --- ParseSubscribe 测试 ---

func TestParseSubscribe(t *testing.T) {
	payload := []byte{
		0x00, 0x01, // packet ID = 1
		0x00, 0x05, // topic length
		't', 'e', 's', 't', '/', // topic "test/"
		0x01, // QoS 1
		0x00, 0x03, // topic length
		'a', '/', 'b', // topic "a/b"
		0x00, // QoS 0
	}

	pkt, err := ParseSubscribe(payload)
	if err != nil {
		t.Fatalf("ParseSubscribe error: %v", err)
	}
	if pkt.PacketID != 1 {
		t.Errorf("expected packetID 1, got %d", pkt.PacketID)
	}
	if len(pkt.Topics) != 2 {
		t.Fatalf("expected 2 topics, got %d", len(pkt.Topics))
	}
	if pkt.Topics[0] != "test/" {
		t.Errorf("expected topic test/, got %s", pkt.Topics[0])
	}
	if pkt.Topics[1] != "a/b" {
		t.Errorf("expected topic a/b, got %s", pkt.Topics[1])
	}
	if pkt.QoS[0] != 1 || pkt.QoS[1] != 0 {
		t.Errorf("unexpected QoS: %v", pkt.QoS)
	}
}

func TestParseSubscribe_TooShort(t *testing.T) {
	_, err := ParseSubscribe([]byte{0x00, 0x01, 0x00})
	if err == nil {
		t.Errorf("expected error for short payload")
	}
}
