package disruptor

import "sync/atomic"

// Sequence 是 Disruptor 中单调递增的并发序号。
type Sequence struct {
	value atomic.Int64
}

// NewSequence 创建指定初始值的序号。
func NewSequence(initial int64) *Sequence {
	sequence := &Sequence{}
	sequence.Store(initial)
	return sequence
}

// Load 读取当前序号。
func (s *Sequence) Load() int64 {
	if s == nil {
		return -1
	}
	return s.value.Load()
}

// Store 写入当前序号。
func (s *Sequence) Store(value int64) {
	if s == nil {
		return
	}
	s.value.Store(value)
}

// CompareAndSwap 以 CAS 方式更新序号。
func (s *Sequence) CompareAndSwap(old int64, next int64) bool {
	if s == nil {
		return false
	}
	return s.value.CompareAndSwap(old, next)
}
