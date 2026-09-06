package testsupport

import (
	"context"
	"errors"
	"sync"
)

// RecordingAppender 以并发安全方式记录测试事件。
type RecordingAppender struct {
	name       string
	mu         sync.Mutex
	events     []Event
	closeCount int
}

// NewRecordingAppender 创建记录型测试 Appender。
func NewRecordingAppender(name string) *RecordingAppender {
	return &RecordingAppender{name: name}
}

// Name 返回 Appender 名称。
func (a *RecordingAppender) Name() string {
	return a.name
}

// Append 记录一条事件。
func (a *RecordingAppender) Append(ctx context.Context, event Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = append(a.events, event)
	return nil
}

// Close 记录关闭次数。
func (a *RecordingAppender) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.closeCount++
	return nil
}

// Events 返回事件快照。
func (a *RecordingAppender) Events() []Event {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]Event(nil), a.events...)
}

// Contains 判断是否记录了指定消息。
func (a *RecordingAppender) Contains(message string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, event := range a.events {
		if event.Message == message {
			return true
		}
	}
	return false
}

// CloseCount 返回关闭次数。
func (a *RecordingAppender) CloseCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.closeCount
}

// FailingAppender 始终返回失败，用于验证降级路径。
type FailingAppender struct {
	name string
}

// NewFailingAppender 创建失败型测试 Appender。
func NewFailingAppender(name string) FailingAppender {
	return FailingAppender{name: name}
}

// Name 返回 Appender 名称。
func (a FailingAppender) Name() string {
	return a.name
}

// Append 返回固定错误。
func (FailingAppender) Append(context.Context, Event) error {
	return errors.New("forced failure")
}

// Close 完成无资源关闭。
func (FailingAppender) Close() error {
	return nil
}
