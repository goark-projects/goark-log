package goarklog

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// BurstFilter 对低优先级日志做令牌桶限流。
type BurstFilter struct {
	level    slog.Level
	rate     float64
	maxBurst float64
	outcome  filterOutcome

	mu     sync.Mutex
	tokens float64
	last   time.Time
}

// NewBurstFilter 创建突发限流过滤器。
func NewBurstFilter(level slog.Level, ratePerSecond float64, maxBurst int, options ...FilterOption) (*BurstFilter, error) {
	if ratePerSecond <= 0 {
		return nil, fmt.Errorf("goark-log: burst filter rate must be > 0")
	}
	if maxBurst <= 0 {
		return nil, fmt.Errorf("goark-log: burst filter maxBurst must be > 0")
	}
	settings := newFilterSettings(FilterNeutral, FilterDeny, options...)
	return &BurstFilter{
		level:    level,
		rate:     ratePerSecond,
		maxBurst: float64(maxBurst),
		outcome:  settings.outcome,
		tokens:   float64(maxBurst),
		last:     time.Now(),
	}, nil
}

func (f *BurstFilter) Decide(_ context.Context, event Event) FilterDecision {
	if f == nil {
		return FilterNeutral
	}
	if event.Level > f.level {
		return FilterNeutral
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(f.last).Seconds()
	f.last = now
	f.tokens += elapsed * f.rate
	if f.tokens > f.maxBurst {
		f.tokens = f.maxBurst
	}
	if f.tokens >= 1 {
		f.tokens--
		return f.outcome.onMatch
	}
	return f.outcome.onMismatch
}
