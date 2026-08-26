package socketappender

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	goarklog "goark.dev/log"
)

const defaultDialTimeout = 5 * time.Second

// Appender 把日志写入 TCP 或 UDP socket。
type Appender struct {
	name         string
	network      string
	address      string
	layout       goarklog.Layout
	dialTimeout  time.Duration
	writeTimeout time.Duration

	mu   sync.Mutex
	conn net.Conn
}

// Option 调整 Socket appender。
type Option func(*Appender)

// WithName 设置 appender 名称。
func WithName(name string) Option {
	return func(appender *Appender) {
		appender.name = name
	}
}

// WithNetwork 设置网络类型。
func WithNetwork(network string) Option {
	return func(appender *Appender) {
		appender.network = network
	}
}

// WithLayout 设置日志布局。
func WithLayout(layout goarklog.Layout) Option {
	return func(appender *Appender) {
		appender.layout = layout
	}
}

// WithDialTimeout 设置连接超时。
func WithDialTimeout(timeout time.Duration) Option {
	return func(appender *Appender) {
		appender.dialTimeout = timeout
	}
}

// WithWriteTimeout 设置写入超时。
func WithWriteTimeout(timeout time.Duration) Option {
	return func(appender *Appender) {
		appender.writeTimeout = timeout
	}
}

// Register 把 Socket appender 注册到指定插件表。
func Register(registry *goarklog.PluginRegistry) error {
	if registry == nil {
		registry = goarklog.DefaultPluginRegistry()
	}
	return registry.RegisterAppender("socket", func(config goarklog.AppenderBuildConfig) (goarklog.Appender, error) {
		options := []Option{
			WithName(config.Name),
			WithNetwork(config.Network),
			WithLayout(config.Layout),
		}
		if timeout, err := parseDuration(config.ConnectTimeout, "connectTimeout"); err != nil {
			return nil, err
		} else if timeout > 0 {
			options = append(options, WithDialTimeout(timeout))
		}
		if timeout, err := parseDuration(config.WriteTimeout, "writeTimeout"); err != nil {
			return nil, err
		} else if timeout > 0 {
			options = append(options, WithWriteTimeout(timeout))
		}
		return New(firstNonBlank(config.Address, config.Target), options...)
	})
}

// New 创建 Socket appender。
func New(address string, options ...Option) (*Appender, error) {
	appender := &Appender{
		name:         "socket",
		network:      "tcp",
		address:      strings.TrimSpace(address),
		layout:       goarklog.JSONLayout{},
		dialTimeout:  defaultDialTimeout,
		writeTimeout: defaultDialTimeout,
	}
	for _, option := range options {
		if option != nil {
			option(appender)
		}
	}
	if strings.TrimSpace(appender.name) == "" {
		return nil, fmt.Errorf("goark-log: socket appender name is empty")
	}
	appender.network = strings.ToLower(strings.TrimSpace(appender.network))
	if appender.network == "" {
		appender.network = "tcp"
	}
	if appender.network != "tcp" && appender.network != "udp" {
		return nil, fmt.Errorf("goark-log: socket appender network %q is unsupported", appender.network)
	}
	if appender.address == "" {
		return nil, fmt.Errorf("goark-log: socket appender address is empty")
	}
	if appender.layout == nil {
		appender.layout = goarklog.JSONLayout{}
	}
	return appender, nil
}

func (a *Appender) Name() string {
	if a == nil || a.name == "" {
		return "socket"
	}
	return a.name
}

func (a *Appender) Append(ctx context.Context, event goarklog.Event) error {
	if a == nil {
		return fmt.Errorf("goark-log: socket appender is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := a.layout.Format(&buf, event); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	conn, err := a.connection()
	if err != nil {
		return err
	}
	if deadline := writeDeadline(a.writeTimeout); !deadline.IsZero() {
		_ = conn.SetWriteDeadline(deadline)
	}
	if _, err := conn.Write(buf.Bytes()); err != nil {
		a.closeLocked()
		return fmt.Errorf("goark-log: socket appender write: %w", err)
	}
	return nil
}

func (a *Appender) Close() error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.closeLocked()
}

func (a *Appender) connection() (net.Conn, error) {
	if a.conn != nil {
		return a.conn, nil
	}
	conn, err := net.DialTimeout(a.network, a.address, a.dialTimeout)
	if err != nil {
		return nil, fmt.Errorf("goark-log: socket appender dial %s %s: %w", a.network, a.address, err)
	}
	a.conn = conn
	return conn, nil
}

func (a *Appender) closeLocked() error {
	if a.conn == nil {
		return nil
	}
	err := a.conn.Close()
	a.conn = nil
	return err
}

func writeDeadline(timeout time.Duration) time.Time {
	if timeout <= 0 {
		return time.Time{}
	}
	return time.Now().Add(timeout)
}

func parseDuration(value string, field string) (time.Duration, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	duration, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("goark-log: invalid socket appender %s %q", field, value)
	}
	return duration, nil
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
