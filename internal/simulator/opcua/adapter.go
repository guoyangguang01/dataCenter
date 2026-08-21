package opcua

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Adapter implements the OPC UA protocol adapter
type Adapter struct {
	endpoint     string
	pollInterval time.Duration
	nodeIDs      []string
	logger       zerolog.Logger

	nodes map[string]*SimNode
	mu    sync.RWMutex
}

// SimNode represents a simulated OPC UA node
type SimNode struct {
	NodeID    string
	Value     float64
	timestamp time.Time
	mu        sync.RWMutex
}

// NewAdapter creates a new OPC UA adapter
func NewAdapter(
	endpoint string,
	pollInterval time.Duration,
	nodeIDs []string,
	logger zerolog.Logger,
) *Adapter {
	if pollInterval == 0 {
		pollInterval = 5 * time.Second
	}
	return &Adapter{
		endpoint:     endpoint,
		pollInterval: pollInterval,
		nodeIDs:      nodeIDs,
		logger:       logger,
		nodes:        make(map[string]*SimNode),
	}
}

// Start starts the OPC UA adapter
func (a *Adapter) Start(ctx context.Context) error {
	a.logger.Info().
		Str("endpoint", a.endpoint).
		Msg("Starting OPC UA simulator")

	// Register nodes
	for _, nodeIDStr := range a.nodeIDs {
		// Store simulated node
		simNode := &SimNode{
			NodeID: nodeIDStr,
			Value:  0.0,
		}
		a.nodes[nodeIDStr] = simNode

		a.logger.Info().
			Str("node_id", nodeIDStr).
			Msg("Registered OPC UA node")
	}

	// Start value update goroutine
	go a.updateLoop(ctx)

	a.logger.Info().
		Str("endpoint", a.endpoint).
		Int("nodes", len(a.nodes)).
		Msg("OPC UA simulator started")

	return nil
}

// updateLoop periodically updates node values
func (a *Adapter) updateLoop(ctx context.Context) {
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.updateNodeValues()
		case <-ctx.Done():
			return
		}
	}
}

// updateNodeValues updates all node values
func (a *Adapter) updateNodeValues() {
	a.mu.RLock()
	defer a.mu.RUnlock()

	now := time.Now()
	for _, simNode := range a.nodes {
		simNode.mu.Lock()

		// Generate a simple oscillating value
		seconds := float64(now.UnixMilli()) / 1000.0
		value := 50.0 + 30.0*math.Sin(seconds/10.0)

		simNode.Value = value
		simNode.timestamp = now

		simNode.mu.Unlock()
	}
}

// Stop stops the OPC UA adapter
func (a *Adapter) Stop() error {
	a.logger.Info().Msg("OPC UA simulator stopped")
	return nil
}

// SendData sends data via OPC UA (updates node values)
func (a *Adapter) SendData(deviceID string, data map[string]interface{}) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Find a node for this device
	for nodeIDStr, simNode := range a.nodes {
		// Update node value based on data
		if val, ok := data["value"]; ok {
			if f, ok := val.(float64); ok {
				simNode.mu.Lock()
				simNode.Value = f
				simNode.mu.Unlock()
			}
		}
		_ = nodeIDStr
	}

	a.logger.Debug().
		Str("device_id", deviceID).
		Int("nodes", len(a.nodes)).
		Msg("Updated OPC UA node values")

	return nil
}

// GetNodeValues returns current node values
func (a *Adapter) GetNodeValues() map[string]float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()

	values := make(map[string]float64)
	for nodeIDStr, simNode := range a.nodes {
		simNode.mu.RLock()
		values[nodeIDStr] = simNode.Value
		simNode.mu.RUnlock()
	}
	return values
}
