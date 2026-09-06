package fileappender

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	internallayout "goark.dev/log/internal/layout"
	"goark.dev/log/internal/logevent"
)

// ConsoleAppender 把日志写入 stdout、stderr 或自定义 writer。
type ConsoleAppender struct {
	name    string
	writer  io.Writer
	layout  Layout
	mu      sync.Mutex
	started bool
	closed  bool
}

// ConsoleOption 调整 ConsoleAppender。
type ConsoleOption func(*ConsoleAppender)

// WithConsoleName 设置 appender 名称。
func WithConsoleName(name string) ConsoleOption {
	return func(appender *ConsoleAppender) {
		appender.name = name
	}
}

// WithConsoleWriter 设置输出 writer，主要用于测试和嵌入式场景。
func WithConsoleWriter(writer io.Writer) ConsoleOption {
	return func(appender *ConsoleAppender) {
		appender.writer = writer
	}
}

// WithConsoleLayout 设置日志布局。
func WithConsoleLayout(layout Layout) ConsoleOption {
	return func(appender *ConsoleAppender) {
		appender.layout = layout
	}
}

// NewConsoleAppender 创建控制台 appender。
func NewConsoleAppender(options ...ConsoleOption) *ConsoleAppender {
	appender := &ConsoleAppender{
		name:   "console",
		writer: os.Stdout,
		layout: internallayout.NewDefaultLayout(),
	}
	for _, option := range options {
		if option != nil {
			option(appender)
		}
	}
	if appender.writer == nil {
		appender.writer = os.Stdout
	}
	if appender.layout == nil {
		appender.layout = internallayout.NewDefaultLayout()
	}
	appender.layout = internallayout.CloneLayout(appender.layout)
	return appender
}

func (a *ConsoleAppender) Name() string {
	if a == nil || a.name == "" {
		return "console"
	}
	return a.name
}

func (a *ConsoleAppender) Append(ctx context.Context, event Event) error {
	if a == nil {
		return fmt.Errorf("goark-log: console appender is nil")
	}
	ctx = logevent.NormalizeContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	if internallayout.RequiresSynchronizedFormatting(a.layout) {
		return a.appendSynchronized(event)
	}
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer releaseBuffer(buf)
	if err := a.layout.Format(buf, event); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return fmt.Errorf("goark-log: console appender %q is closed", a.Name())
	}
	if err := a.ensureStartedLocked(); err != nil {
		return err
	}
	_, err := a.writer.Write(buf.Bytes())
	return err
}

func (a *ConsoleAppender) appendSynchronized(event Event) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return fmt.Errorf("goark-log: console appender %q is closed", a.Name())
	}
	if err := a.ensureStartedLocked(); err != nil {
		return err
	}
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer releaseBuffer(buf)
	if err := a.layout.Format(buf, event); err != nil {
		return err
	}
	_, err := a.writer.Write(buf.Bytes())
	return err
}

func (a *ConsoleAppender) ensureStartedLocked() error {
	if a.started {
		return nil
	}
	if _, err := internallayout.WriteHeader(a.writer, a.layout); err != nil {
		return err
	}
	a.started = true
	return nil
}

func (a *ConsoleAppender) Close() error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return nil
	}
	a.closed = true
	if !a.started {
		return nil
	}
	_, err := internallayout.WriteFooter(a.writer, a.layout)
	return err
}
