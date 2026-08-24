package rule

import (
	"sync"
	"testing"
	"time"
)

func TestPipelineState_SetGet(t *testing.T) {
	state := NewPipelineState()

	// 获取不存在的 key
	val, ok := state.Get("missing")
	if ok {
		t.Errorf("expected ok=false for missing key, got true")
	}
	if val != nil {
		t.Errorf("expected nil for missing key, got %v", val)
	}

	// 设置并获取
	state.Set("temperature", 25.6)
	val, ok = state.Get("temperature")
	if !ok {
		t.Errorf("expected ok=true")
	}
	if val != 25.6 {
		t.Errorf("expected 25.6, got %v", val)
	}

	// 覆盖写入
	state.Set("temperature", 30.0)
	val, _ = state.Get("temperature")
	if val != 30.0 {
		t.Errorf("expected 30.0 after overwrite, got %v", val)
	}
}

func TestPipelineState_Concurrent(t *testing.T) {
	state := NewPipelineState()
	var wg sync.WaitGroup

	// 并发写入
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			state.Set("key", n)
		}(i)
	}

	// 并发读取
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			state.Get("key")
		}()
	}

	wg.Wait()
	// 不 panic 就算通过
}

func TestPipelineState_GetWindow(t *testing.T) {
	state := NewPipelineState()

	// 第一次获取会创建
	w1 := state.GetWindow("temp", 10, 1*time.Minute)
	if w1 == nil {
		t.Fatal("expected non-nil window")
	}

	// 第二次获取返回同一个实例
	w2 := state.GetWindow("temp", 10, 1*time.Minute)
	if w1 != w2 {
		t.Errorf("expected same window instance")
	}

	// 不同 key 返回不同窗口
	w3 := state.GetWindow("humidity", 5, 30*time.Second)
	if w1 == w3 {
		t.Errorf("expected different window for different key")
	}
}

func TestSlidingWindow_AddAndGetValues(t *testing.T) {
	w := NewSlidingWindow(5, 1*time.Minute)

	w.Add(1.0)
	w.Add(2.0)
	w.Add(3.0)

	vals := w.GetValues()
	if len(vals) != 3 {
		t.Fatalf("expected 3 values, got %d", len(vals))
	}
	if vals[0] != 1.0 || vals[1] != 2.0 || vals[2] != 3.0 {
		t.Errorf("unexpected values: %v", vals)
	}
}

func TestSlidingWindow_MaxSize(t *testing.T) {
	w := NewSlidingWindow(3, 1*time.Minute)

	w.Add(1)
	w.Add(2)
	w.Add(3)
	w.Add(4)
	w.Add(5)

	if w.Count() != 3 {
		t.Errorf("expected count 3, got %d", w.Count())
	}

	vals := w.GetValues()
	// 应该保留最后 3 个
	if vals[0] != 3 || vals[1] != 4 || vals[2] != 5 {
		t.Errorf("expected [3 4 5], got %v", vals)
	}
}

func TestSlidingWindow_TimeExpiry(t *testing.T) {
	w := NewSlidingWindow(100, 50*time.Millisecond)

	w.Add("old")
	time.Sleep(80 * time.Millisecond)
	w.Add("new")

	// "old" 应该过期了
	if w.Count() != 1 {
		t.Errorf("expected 1 item after expiry, got %d", w.Count())
	}
	vals := w.GetValues()
	if len(vals) != 1 || vals[0] != "new" {
		t.Errorf("expected [new], got %v", vals)
	}
}

func TestSlidingWindow_Clear(t *testing.T) {
	w := NewSlidingWindow(10, 1*time.Minute)
	w.Add(1)
	w.Add(2)
	w.Clear()

	if w.Count() != 0 {
		t.Errorf("expected 0 after clear, got %d", w.Count())
	}
}

func TestSlidingWindow_Empty(t *testing.T) {
	w := NewSlidingWindow(10, 1*time.Minute)

	if w.Count() != 0 {
		t.Errorf("expected 0 for empty window, got %d", w.Count())
	}
	vals := w.GetValues()
	if len(vals) != 0 {
		t.Errorf("expected empty slice, got %v", vals)
	}
}
