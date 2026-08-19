package mqtt

import (
    "encoding/binary"
    "fmt"
    "io"
)

const (
    CONNECT     byte = 0x10
    CONNACK     byte = 0x20
    PUBLISH     byte = 0x30
    PUBACK      byte = 0x40
    SUBSCRIBE   byte = 0x80
    SUBACK      byte = 0x90
    UNSUBSCRIBE byte = 0xA0
    UNSUBACK    byte = 0xB0
    PINGREQ     byte = 0xC0
    PINGRESP    byte = 0xD0
    DISCONNECT  byte = 0xE0
)

type Packet struct {
    Type    byte
    Flags   byte
    Payload []byte
}

func ReadPacket(r io.Reader) (*Packet, error) {
    var header [1]byte
    if _, err := io.ReadFull(r, header[:]); err != nil {
        return nil, err
    }
    typeFlags := header[0]
    msgType := typeFlags & 0xF0
    flags := typeFlags & 0x0F

    var multiplier int = 1
    var remainingLength int
    for {
        if _, err := io.ReadFull(r, header[:]); err != nil {
            return nil, err
        }
        remainingLength += int(header[0]&0x7F) * multiplier
        if header[0]&0x80 == 0 {
            break
        }
        multiplier *= 128
    }

    payload := make([]byte, remainingLength)
    if remainingLength > 0 {
        if _, err := io.ReadFull(r, payload); err != nil {
            return nil, err
        }
    }
    return &Packet{Type: msgType, Flags: flags, Payload: payload}, nil
}

func WritePacket(w io.Writer, pkt *Packet) error {
    typeFlags := pkt.Type | (pkt.Flags & 0x0F)
    if _, err := w.Write([]byte{typeFlags}); err != nil {
        return err
    }

    remaining := len(pkt.Payload)
    for {
        encodedByte := byte(remaining % 128)
        remaining = remaining / 128
        if remaining > 0 {
            encodedByte |= 0x80
        }
        if _, err := w.Write([]byte{encodedByte}); err != nil {
            return err
        }
        if remaining == 0 {
            break
        }
    }

    if len(pkt.Payload) > 0 {
        if _, err := w.Write(pkt.Payload); err != nil {
            return err
        }
    }
    return nil
}

type ConnectPacket struct {
    ProtocolName  string
    ProtocolLevel byte
    ConnectFlags  byte
    KeepAlive     uint16
    ClientID      string
    Username      string
    Password      []byte
    WillTopic     string
    WillMessage   []byte
}

func ParseConnect(payload []byte) (*ConnectPacket, error) {
    if len(payload) < 10 {
        return nil, fmt.Errorf("CONNECT payload too short")
    }
    p := &ConnectPacket{}
    offset := 0
    nameLen := int(binary.BigEndian.Uint16(payload[offset:]))
    offset += 2
    p.ProtocolName = string(payload[offset : offset+nameLen])
    offset += nameLen
    p.ProtocolLevel = payload[offset]
    offset++
    p.ConnectFlags = payload[offset]
    offset++
    p.KeepAlive = binary.BigEndian.Uint16(payload[offset:])
    offset += 2
    if len(payload) > offset {
        idLen := int(binary.BigEndian.Uint16(payload[offset:]))
        offset += 2
        p.ClientID = string(payload[offset : offset+idLen])
        offset += idLen
    }
    return p, nil
}

type PublishPacket struct {
    TopicName string
    PacketID  uint16
    Payload   []byte
}

func ParsePublish(flags byte, payload []byte) (*PublishPacket, error) {
    if len(payload) < 4 {
        return nil, fmt.Errorf("PUBLISH payload too short")
    }
    p := &PublishPacket{}
    offset := 0
    topicLen := int(binary.BigEndian.Uint16(payload[offset:]))
    offset += 2
    p.TopicName = string(payload[offset : offset+topicLen])
    offset += topicLen
    if flags&0x06 != 0 {
        p.PacketID = binary.BigEndian.Uint16(payload[offset:])
        offset += 2
    }
    p.Payload = payload[offset:]
    return p, nil
}

type SubscribePacket struct {
    PacketID uint16
    Topics   []string
    QoS      []byte
}

func ParseSubscribe(payload []byte) (*SubscribePacket, error) {
    if len(payload) < 5 {
        return nil, fmt.Errorf("SUBSCRIBE payload too short")
    }
    p := &SubscribePacket{}
    offset := 0
    p.PacketID = binary.BigEndian.Uint16(payload[offset:])
    offset += 2
    for offset < len(payload) {
        topicLen := int(binary.BigEndian.Uint16(payload[offset:]))
        offset += 2
        topic := string(payload[offset : offset+topicLen])
        offset += topicLen
        qos := payload[offset]
        offset++
        p.Topics = append(p.Topics, topic)
        p.QoS = append(p.QoS, qos)
    }
    return p, nil
}
