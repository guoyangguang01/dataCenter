package nats

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Config struct {
	URL              string        `yaml:"url"`
	MaxReconnect     int           `yaml:"max_reconnect"`
	ReconnectWait    time.Duration `yaml:"reconnect_wait"`
	ReconnectBufSize int           `yaml:"reconnect_buf_size"`
}

func DefaultConfig() Config {
	return Config{
		URL:              "nats://localhost:4222",
		MaxReconnect:     -1,
		ReconnectWait:    2 * time.Second,
		ReconnectBufSize: 1024 * 1024,
	}
}

type Client struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	config Config
}

func New(config Config) (*Client, error) {
	opts := []nats.Option{
		nats.MaxReconnects(config.MaxReconnect),
		nats.ReconnectWait(config.ReconnectWait),
		nats.ReconnectBufSize(config.ReconnectBufSize),
	}

	nc, err := nats.Connect(config.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream: %w", err)
	}

	fmt.Println("[NATS] connected to", nc.ConnectedUrl())
	return &Client{nc: nc, js: js, config: config}, nil
}

func (c *Client) JetStream() jetstream.JetStream { return c.js }
func (c *Client) Conn() *nats.Conn               { return c.nc }

func (c *Client) Close() {
	if c.nc != nil {
		c.nc.Close()
		fmt.Println("[NATS] connection closed")
	}
}

func (c *Client) IsConnected() bool {
	return c.nc != nil && c.nc.IsConnected()
}
