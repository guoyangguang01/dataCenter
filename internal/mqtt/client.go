package mqtt

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/datacenter/internal/gateway"
)

type Client struct {
	conn      net.Conn
	codec     *Codec
	sessions  *SessionManager
	publisher gateway.Publisher
	onData    DeviceStatusCallback
	clientID  string
	username  string
	connected bool
	keepAlive uint16
}

func NewClient(conn net.Conn, codec *Codec, sessions *SessionManager, publisher gateway.Publisher) *Client {
	return &Client{
		conn:      conn,
		codec:     codec,
		sessions:  sessions,
		publisher: publisher,
	}
}

func (c *Client) ReadLoop() error {
	for {
		pkt, err := ReadPacket(c.conn)
		if err != nil {
			return err
		}

		switch pkt.Type {
		case CONNECT:
			if err := c.handleConnect(pkt); err != nil {
				return err
			}
		case PUBLISH:
			if err := c.handlePublish(pkt); err != nil {
				fmt.Println("[MQTT] publish error: ", err)
			}
		case SUBSCRIBE:
			if err := c.handleSubscribe(pkt); err != nil {
				return err
			}
		case PINGREQ:
			if err := c.handlePingreq(); err != nil {
				return err
			}
		case DISCONNECT:
			c.handleDisconnect()
			return nil
		default:
			fmt.Println("[MQTT] unhandled packet type: 0x%02X", pkt.Type)
		}
	}
}

func (c *Client) handleConnect(pkt *Packet) error {
	connPkt, err := ParseConnect(pkt.Payload)
	if err != nil {
		return fmt.Errorf("failed to parse CONNECT: %w", err)
	}

	c.clientID = connPkt.ClientID
	c.username = connPkt.Username
	c.keepAlive = connPkt.KeepAlive

	session := &Session{
		DeviceID:     c.clientID,
		ClientID:     c.clientID,
		ConnectedAt:  time.Now(),
		LastSeen:     time.Now(),
		CleanSession: connPkt.ConnectFlags&0x02 != 0,
		KeepAlive:    connPkt.KeepAlive,
	}
	c.sessions.Set(c.clientID, session)
	c.connected = true

	connack := &Packet{Type: CONNACK, Payload: []byte{0x00, 0x00}}
	if err := WritePacket(c.conn, connack); err != nil {
		return err
	}

	fmt.Println("[MQTT] client connected:", c.clientID)
	return nil
}

func (c *Client) handlePublish(pkt *Packet) error {
	pubPkt, err := ParsePublish(pkt.Flags, pkt.Payload)
	if err != nil {
		fmt.Printf("[MQTT] publish parse error: %v\n", err)
		return fmt.Errorf("failed to parse PUBLISH: %w", err)
	}

	fmt.Printf("[MQTT] received PUBLISH from %s topic=%s\n", c.clientID, pubPkt.TopicName)

	c.sessions.UpdateLastSeen(c.clientID)

	// 通知设备在线（clientID 即设备 ID）
	if c.onData != nil {
		c.onData(c.clientID)
	}

	env := c.codec.ToEnvelope(c.clientID, pubPkt.TopicName, pubPkt.Payload)
	if err := c.publisher.PublishEnvelope(env); err != nil {
		return fmt.Errorf("failed to publish envelope: %w", err)
	}

	fmt.Println("[MQTT] published:", c.clientID, pubPkt.TopicName)
	return nil
}

func (c *Client) handleSubscribe(pkt *Packet) error {
	subPkt, err := ParseSubscribe(pkt.Payload)
	if err != nil {
		return fmt.Errorf("failed to parse SUBSCRIBE: %w", err)
	}

	session, ok := c.sessions.Get(c.clientID)
	if ok {
		session.Subscriptions = append(session.Subscriptions, subPkt.Topics...)
	}

	granted := make([]byte, len(subPkt.QoS))
	for i, qos := range subPkt.QoS {
		granted[i] = qos
	}

	suback := &Packet{Type: SUBACK, Payload: append([]byte{byte(subPkt.PacketID >> 8), byte(subPkt.PacketID)}, granted...)}
	if err := WritePacket(c.conn, suback); err != nil {
		return err
	}

	fmt.Println("[MQTT] subscribed:", c.clientID)
	return nil
}

func (c *Client) handlePingreq() error {
	pingresp := &Packet{Type: PINGRESP}
	return WritePacket(c.conn, pingresp)
}

func (c *Client) handleDisconnect() {
	c.sessions.Remove(c.clientID)
	c.connected = false
	fmt.Println("[MQTT] client disconnected: ", c.clientID)
}

func (c *Client) SendCommand(topic string, payload []byte) error {
	if !c.connected {
		return fmt.Errorf("client not connected")
	}
	pubPkt := &Packet{
		Type:    PUBLISH,
		Payload: buildPublishPayload(topic, payload),
	}
	return WritePacket(c.conn, pubPkt)
}

func buildPublishPayload(topic string, payload []byte) []byte {
	topicBytes := []byte(topic)
	result := make([]byte, 2+len(topicBytes)+len(payload))
	result[0] = byte(len(topicBytes) >> 8)
	result[1] = byte(len(topicBytes))
	copy(result[2:2+len(topicBytes)], topicBytes)
	copy(result[2+len(topicBytes):], payload)
	return result
}

func (c *Client) MatchesFilter(filter string) bool {
	if filter == "" {
		return true
	}
	return strings.Contains(filter, c.clientID)
}
