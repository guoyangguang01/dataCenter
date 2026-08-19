package rule

import (
	"sync"
	"time"
)

// PipelineState 规则链执行状态
type PipelineState struct {
	mu      sync.RWMutex
	data    map[string]interface{}
	windows map[string]*SlidingWindow
}

// NewPipelineState 创建新的管道状态
func NewPipelineState() *PipelineState {
	return &PipelineState{
		data:    make(map[string]interface{}),
		windows: make(map[string]*SlidingWindow),
	}
}

// Set 设置状态值
func (s *PipelineState) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

// Get 获取状态值
func (s *PipelineState) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

// GetWindow 获取滑动窗口
func (s *PipelineState) GetWindow(key string, maxSize int, windowSize time.Duration) *SlidingWindow {
	s.mu.Lock()
	defer s.mu.Unlock()

	if w, ok := s.windows[key]; ok {
		return w
	}
	w := NewSlidingWindow(maxSize, windowSize)
	s.windows[key] = w
	return w
}

// SlidingWindow 滑动窗口
type SlidingWindow struct {
	mu         sync.RWMutex
	items      []WindowItem
	maxSize    int
	windowSize time.Duration
}

type WindowItem struct {
	Value     interface{}
	Timestamp time.Time
}

// NewSlidingWindow 创建滑动窗口
func NewSlidingWindow(maxSize int, windowSize time.Duration) *SlidingWindow {
	return &SlidingWindow{
		items:      make([]WindowItem, 0, maxSize),
		maxSize:    maxSize,
		windowSize: windowSize,
	}
}

// Add 添加数据到窗口
func (w *SlidingWindow) Add(value interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	w.items = append(w.items, WindowItem{Value: value, Timestamp: now})

	// 移除过期数据
	cutoff := now.Add(-w.windowSize)
	idx := 0
	for idx < len(w.items) && w.items[idx].Timestamp.Before(cutoff) {
		idx++
	}
	if idx > 0 {
		w.items = w.items[idx:]
	}

	// 超出最大大小，移除最旧的
	if len(w.items) > w.maxSize {
		w.items = w.items[len(w.items)-w.maxSize:]
	}
}

// GetValues 获取窗口内所有值
func (w *SlidingWindow) GetValues() []interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	values := make([]interface{}, len(w.items))
	for i, item := range w.items {
		values[i] = item.Value
	}
	return values
}

// Count 窗口内数据数量
func (w *SlidingWindow) Count() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.items)
}

// Clear 清空窗口
func (w *SlidingWindow) Clear() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.items = w.items[:0]
}
