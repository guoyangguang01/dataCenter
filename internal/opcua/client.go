package opcua

import (
	"context"
	"fmt"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

// WriteRequest 写入请求
type WriteRequest struct {
	NodeID string      `json:"node_id"` // 目标节点 ID，如 "ns=2;s=Temperature"
	Value  interface{} `json:"value"`   // 要写入的值
}

// Client OPC UA 客户端封装
type Client struct {
	conn *opcua.Client
	ctx  context.Context
}

// Connect 连接到 OPC UA Server
func Connect(endpoint string) (*Client, error) {
	ctx := context.Background()

	opts := []opcua.Option{
		opcua.SecurityMode(ua.MessageSecurityModeNone),
		opcua.RequestTimeout(10 * time.Second),
	}

	c, err := opcua.NewClient(endpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OPC UA client: %w", err)
	}
	if err := c.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to OPC UA server: %w", err)
	}

	fmt.Println("[OPC UA] connected to", endpoint)
	return &Client{conn: c, ctx: ctx}, nil
}

// ReadNodes 批量读取节点值
func (c *Client) ReadNodes(nodeIDs []string) (map[string]interface{}, error) {
	ids := make([]*ua.ReadValueID, len(nodeIDs))
	for i, nid := range nodeIDs {
		ids[i] = &ua.ReadValueID{NodeID: ua.NewStringNodeID(0, nid)}
	}

	req := &ua.ReadRequest{
		NodesToRead: ids,
		TimestampsToReturn: ua.TimestampsToReturnBoth,
	}

	resp, err := c.conn.Read(c.ctx, req)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}

	if resp.Results == nil || len(resp.Results) != len(nodeIDs) {
		return nil, fmt.Errorf("unexpected response length")
	}

	result := make(map[string]interface{}, len(nodeIDs))
	for i, res := range resp.Results {
		if res.Status != ua.StatusGood {
			fmt.Printf("[OPC UA] node %s status: %v\n", nodeIDs[i], res.Status)
			continue
		}
		val := res.Value.Value()
		result[nodeIDs[i]] = val
	}

	return result, nil
}

// WriteNode 向单个节点写入值
func (c *Client) WriteNode(nodeID string, value interface{}) error {
	node := ua.NewStringNodeID(0, nodeID)

	// 将值转换为 ua.Variant
	variant, err := ua.NewVariant(value)
	if err != nil {
		return fmt.Errorf("failed to create variant for node %s: %w", nodeID, err)
	}

	req := &ua.WriteRequest{
		NodesToWrite: []*ua.WriteValue{
			{
				NodeID:      node,
				AttributeID: ua.AttributeIDValue,
				Value: &ua.DataValue{
					Value:        variant,
					EncodingMask: ua.DataValueValue,
				},
			},
		},
	}

	resp, err := c.conn.Write(c.ctx, req)
	if err != nil {
		return fmt.Errorf("write failed for node %s: %w", nodeID, err)
	}

	if resp.Results != nil && len(resp.Results) > 0 {
		if resp.Results[0] != ua.StatusGood {
			return fmt.Errorf("write to node %s returned status: %v", nodeID, resp.Results[0])
		}
	}

	fmt.Printf("[OPC UA] ✅ 写入成功: node=%s value=%v\n", nodeID, value)
	return nil
}

// Close 关闭连接
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close(context.Background())
	}
}
