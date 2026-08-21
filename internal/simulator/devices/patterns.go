package devices

import (
	"math"
	"math/rand"
	"time"
)

// Pattern defines the interface for data generation patterns
type Pattern interface {
	// Next generates the next value at the given time
	Next(t time.Time) float64
}

// PatternConfig holds configuration for a pattern
type PatternConfig struct {
	Type             string  `yaml:"type"`
	Amplitude        float64 `yaml:"amplitude,omitempty"`
	Period           float64 `yaml:"period,omitempty"`
	Offset           float64 `yaml:"offset,omitempty"`
	Phase            float64 `yaml:"phase,omitempty"`
	Noise            float64 `yaml:"noise,omitempty"`
	Min              float64 `yaml:"min,omitempty"`
	Max              float64 `yaml:"max,omitempty"`
	Step             float64 `yaml:"step,omitempty"`
	Base             float64 `yaml:"base,omitempty"`
	BaseValue        float64 `yaml:"base_value,omitempty"`
	PeakValue        float64 `yaml:"peak_value,omitempty"`
	Duration         float64 `yaml:"duration,omitempty"`
	Interval         float64 `yaml:"interval,omitempty"`
	NoiseAmplitude   float64 `yaml:"noise_amplitude,omitempty"`
	Values           []float64 `yaml:"values,omitempty"`
	StepInterval     float64 `yaml:"step_interval,omitempty"`
}

// NewPattern creates a new pattern from configuration
func NewPattern(cfg PatternConfig) Pattern {
	switch cfg.Type {
	case "sine":
		return NewSinePattern(cfg)
	case "random_walk":
		return NewRandomWalkPattern(cfg)
	case "step":
		return NewStepPattern(cfg)
	case "pulse":
		return NewPulsePattern(cfg)
	case "constant_noise":
		return NewConstantNoisePattern(cfg)
	default:
		return NewConstantNoisePattern(PatternConfig{Base: 0, NoiseAmplitude: 0.1})
	}
}

// SinePattern generates sine wave values with optional noise
type SinePattern struct {
	amplitude float64
	period    float64
	offset    float64
	phase     float64
	noise     float64
	rng       *rand.Rand
}

// NewSinePattern creates a new sine wave pattern
func NewSinePattern(cfg PatternConfig) *SinePattern {
	period := cfg.Period
	if period <= 0 {
		period = 60.0 // Default 60 seconds
	}
	return &SinePattern{
		amplitude: cfg.Amplitude,
		period:    period,
		offset:    cfg.Offset,
		phase:     cfg.Phase,
		noise:     cfg.Noise,
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Next generates the next sine wave value
func (p *SinePattern) Next(t time.Time) float64 {
	seconds := float64(t.UnixMilli()) / 1000.0
	angle := 2 * math.Pi * seconds / p.period
	value := p.amplitude*math.Sin(angle+p.phase) + p.offset
	if p.noise > 0 {
		value += (p.rng.Float64()*2 - 1) * p.noise
	}
	return value
}

// RandomWalkPattern generates random walk values
type RandomWalkPattern struct {
	min       float64
	max       float64
	step      float64
	current   float64
	rng       *rand.Rand
	initialized bool
}

// NewRandomWalkPattern creates a new random walk pattern
func NewRandomWalkPattern(cfg PatternConfig) *RandomWalkPattern {
	initial := (cfg.Min + cfg.Max) / 2
	return &RandomWalkPattern{
		min:     cfg.Min,
		max:     cfg.Max,
		step:    cfg.Step,
		current: initial,
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Next generates the next random walk value
func (p *RandomWalkPattern) Next(t time.Time) float64 {
	delta := (p.rng.Float64()*2 - 1) * p.step
	p.current += delta

	// Clamp to min/max range
	if p.current < p.min {
		p.current = p.min + p.step
	} else if p.current > p.max {
		p.current = p.max - p.step
	}

	return p.current
}

// StepPattern generates step function values
type StepPattern struct {
	values    []float64
	interval  time.Duration
	rng       *rand.Rand
	startTime time.Time
}

// NewStepPattern creates a new step pattern
func NewStepPattern(cfg PatternConfig) *StepPattern {
	values := cfg.Values
	if len(values) == 0 {
		values = []float64{0, 1}
	}
	interval := time.Duration(cfg.StepInterval * float64(time.Second))
	if interval <= 0 {
		interval = 10 * time.Second
	}
	return &StepPattern{
		values:   values,
		interval: interval,
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Next generates the next step value
func (p *StepPattern) Next(t time.Time) float64 {
	if p.startTime.IsZero() {
		p.startTime = t
	}
	elapsed := t.Sub(p.startTime)
	index := int(elapsed/p.interval) % len(p.values)
	return p.values[index]
}

// PulsePattern generates pulse/burst values
type PulsePattern struct {
	baseValue  float64
	peakValue  float64
	duration   time.Duration
	interval   time.Duration
	rng        *rand.Rand
	startTime  time.Time
}

// NewPulsePattern creates a new pulse pattern
func NewPulsePattern(cfg PatternConfig) *PulsePattern {
	duration := time.Duration(cfg.Duration * float64(time.Second))
	if duration <= 0 {
		duration = 1 * time.Second
	}
	interval := time.Duration(cfg.Interval * float64(time.Second))
	if interval <= 0 {
		interval = 10 * time.Second
	}
	return &PulsePattern{
		baseValue: cfg.BaseValue,
		peakValue: cfg.PeakValue,
		duration:  duration,
		interval:  interval,
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Next generates the next pulse value
func (p *PulsePattern) Next(t time.Time) float64 {
	if p.startTime.IsZero() {
		p.startTime = t
	}
	elapsed := t.Sub(p.startTime)
	cyclePosition := elapsed % p.interval

	if cyclePosition < p.duration {
		return p.peakValue
	}
	return p.baseValue
}

// ConstantNoisePattern generates constant value with noise
type ConstantNoisePattern struct {
	base   float64
	noise  float64
	rng    *rand.Rand
}

// NewConstantNoisePattern creates a new constant noise pattern
func NewConstantNoisePattern(cfg PatternConfig) *ConstantNoisePattern {
	return &ConstantNoisePattern{
		base:  cfg.Base,
		noise: cfg.NoiseAmplitude,
		rng:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Next generates the next constant noise value
func (p *ConstantNoisePattern) Next(t time.Time) float64 {
	if p.noise > 0 {
		return p.base + (p.rng.Float64()*2-1)*p.noise
	}
	return p.base
}
