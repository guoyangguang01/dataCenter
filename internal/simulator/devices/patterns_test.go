package devices

import (
	"math"
	"testing"
	"time"
)

func TestSinePattern(t *testing.T) {
	cfg := PatternConfig{
		Type:      "sine",
		Amplitude: 10.0,
		Period:    60.0,
		Offset:    20.0,
		Phase:     0,
		Noise:     0,
	}

	pattern := NewSinePattern(cfg)
	now := time.Now()

	// Generate values and check they oscillate
	values := make([]float64, 100)
	for i := range values {
		values[i] = pattern.Next(now.Add(time.Duration(i) * time.Second))
	}

	// Check that values are within expected range [offset-amplitude, offset+noise]
	for i, val := range values {
		if val < 10.0-0.01 || val > 30.0+0.01 {
			t.Errorf("Value %f out of expected range [10, 30] at step %d", val, i)
		}
	}

	// Check that values vary (sine wave oscillates)
	minVal, maxVal := values[0], values[0]
	for _, val := range values {
		if val < minVal {
			minVal = val
		}
		if val > maxVal {
			maxVal = val
		}
	}
	if maxVal-minVal < 1.0 {
		t.Errorf("Expected values to oscillate, but range is only %f", maxVal-minVal)
	}
}

func TestSinePatternWithNoise(t *testing.T) {
	cfg := PatternConfig{
		Type:      "sine",
		Amplitude: 10.0,
		Period:    60.0,
		Offset:    20.0,
		Phase:     0,
		Noise:     2.0,
	}

	pattern := NewSinePattern(cfg)
	now := time.Now()

	// Generate multiple values, check they vary
	values := make([]float64, 10)
	for i := range values {
		values[i] = pattern.Next(now.Add(time.Duration(i) * time.Second))
	}

	// Check that values are not all identical
	allSame := true
	for i := 1; i < len(values); i++ {
		if math.Abs(values[i]-values[0]) > 0.001 {
			allSame = false
			break
		}
	}
	if allSame {
		t.Error("Expected values to vary with noise, but all are identical")
	}
}

func TestRandomWalkPattern(t *testing.T) {
	cfg := PatternConfig{
		Type: "random_walk",
		Min:  10.0,
		Max:  20.0,
		Step: 1.0,
	}

	pattern := NewRandomWalkPattern(cfg)
	now := time.Now()

	// Generate 100 values
	for i := 0; i < 100; i++ {
		val := pattern.Next(now.Add(time.Duration(i) * time.Second))
		if val < cfg.Min || val > cfg.Max {
			t.Errorf("Value %f out of range [%f, %f] at step %d", val, cfg.Min, cfg.Max, i)
		}
	}
}

func TestStepPattern(t *testing.T) {
	cfg := PatternConfig{
		Type:         "step",
		Values:       []float64{0, 1, 2},
		StepInterval: 10.0, // 10 seconds
	}

	pattern := NewStepPattern(cfg)
	now := time.Now()

	// First value should be values[0]
	val := pattern.Next(now)
	if val != 0 {
		t.Errorf("Expected 0 at start, got %f", val)
	}

	// After 5 seconds, still values[0]
	val = pattern.Next(now.Add(5 * time.Second))
	if val != 0 {
		t.Errorf("Expected 0 at 5s, got %f", val)
	}

	// After 10 seconds, should be values[1]
	val = pattern.Next(now.Add(10 * time.Second))
	if val != 1 {
		t.Errorf("Expected 1 at 10s, got %f", val)
	}

	// After 20 seconds, should be values[2]
	val = pattern.Next(now.Add(20 * time.Second))
	if val != 2 {
		t.Errorf("Expected 2 at 20s, got %f", val)
	}

	// After 30 seconds, should wrap to values[0]
	val = pattern.Next(now.Add(30 * time.Second))
	if val != 0 {
		t.Errorf("Expected 0 at 30s (wrap), got %f", val)
	}
}

func TestPulsePattern(t *testing.T) {
	cfg := PatternConfig{
		Type:      "pulse",
		BaseValue: 0.0,
		PeakValue: 100.0,
		Duration:  5.0, // 5 seconds
		Interval:  60.0, // 60 seconds
	}

	pattern := NewPulsePattern(cfg)
	now := time.Now()

	// At start, should be peak (in pulse duration)
	val := pattern.Next(now)
	if val != 100.0 {
		t.Errorf("Expected 100.0 at start, got %f", val)
	}

	// After 3 seconds, still in pulse
	val = pattern.Next(now.Add(3 * time.Second))
	if val != 100.0 {
		t.Errorf("Expected 100.0 at 3s, got %f", val)
	}

	// After 6 seconds, should be base
	val = pattern.Next(now.Add(6 * time.Second))
	if val != 0.0 {
		t.Errorf("Expected 0.0 at 6s, got %f", val)
	}

	// After 60 seconds, should be peak again
	val = pattern.Next(now.Add(60 * time.Second))
	if val != 100.0 {
		t.Errorf("Expected 100.0 at 60s, got %f", val)
	}
}

func TestConstantNoisePattern(t *testing.T) {
	cfg := PatternConfig{
		Type:           "constant_noise",
		Base:           50.0,
		NoiseAmplitude: 5.0,
	}

	pattern := NewConstantNoisePattern(cfg)
	now := time.Now()

	// Generate 100 values
	for i := 0; i < 100; i++ {
		val := pattern.Next(now.Add(time.Duration(i) * time.Second))
		if val < 45.0 || val > 55.0 {
			t.Errorf("Value %f out of expected range [45, 55] at step %d", val, i)
		}
	}
}

func TestNewPattern(t *testing.T) {
	tests := []struct {
		name string
		cfg  PatternConfig
	}{
		{"sine", PatternConfig{Type: "sine"}},
		{"random_walk", PatternConfig{Type: "random_walk"}},
		{"step", PatternConfig{Type: "step"}},
		{"pulse", PatternConfig{Type: "pulse"}},
		{"constant_noise", PatternConfig{Type: "constant_noise"}},
		{"unknown", PatternConfig{Type: "unknown"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := NewPattern(tt.cfg)
			if pattern == nil {
				t.Errorf("NewPattern(%s) returned nil", tt.name)
			}
		})
	}
}
