package syslogappender

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	goarklog "goark.dev/log"
)

const (
	defaultFacility = 16
	defaultNetwork  = "udp"
)

// Appender 把事件发送到 RFC5424 风格 syslog 端点。
type Appender struct {
	name         string
	network      string
	address      string
	facility     int
	facilityName string
	appName      string
	hostname     string
	layout       goarklog.Layout
	dialTimeout  time.Duration
	writeTimeout time.Duration

	mu   sync.Mutex
	conn net.Conn
}

// Option 调整 Syslog appender。
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

// WithFacility 设置 syslog facility。
func WithFacility(facility string) Option {
	return func(appender *Appender) {
		appender.facilityName = facility
	}
}

// WithAppName 设置 syslog APP-NAME。
func WithAppName(appName string) Option {
	return func(appender *Appender) {
		appender.appName = appName
	}
}

// WithLayout 设置 syslog MSG 的内部布局。
func WithLayout(layout goarklog.Layout) Option {
	return func(appender *Appender) {
		appender.layout = layout
	}
}

// Register 把 Syslog appender 注册到指定插件表。
func Register(registry *goarklog.PluginRegistry) error {
	if registry == nil {
		registry = goarklog.DefaultPluginRegistry()
	}
	return registry.RegisterAppender("syslog", func(config goarklog.AppenderBuildConfig) (goarklog.Appender, error) {
		return New(firstNonBlank(config.Address, config.Target),
			WithName(config.Name),
			WithNetwork(config.Network),
			WithFacility(config.Facility),
			WithAppName(config.AppName),
			WithLayout(config.Layout),
		)
	})
}

// New 创建 Syslog appender。
func New(address string, options ...Option) (*Appender, error) {
	hostname, _ := os.Hostname()
	appender := &Appender{
		name:         "syslog",
		network:      defaultNetwork,
		address:      strings.TrimSpace(address),
		facility:     defaultFacility,
		appName:      "goark-log",
		hostname:     hostname,
		layout:       goarklog.TextLayout{},
		dialTimeout:  5 * time.Second,
		writeTimeout: 5 * time.Second,
	}
	for _, option := range options {
		if option != nil {
			option(appender)
		}
	}
	if strings.TrimSpace(appender.name) == "" {
		return nil, fmt.Errorf("goark-log: syslog appender name is empty")
	}
	appender.network = strings.ToLower(strings.TrimSpace(appender.network))
	if appender.network == "" {
		appender.network = defaultNetwork
	}
	if appender.network != "udp" && appender.network != "tcp" {
		return nil, fmt.Errorf("goark-log: syslog appender network %q is unsupported", appender.network)
	}
	if appender.address == "" {
		return nil, fmt.Errorf("goark-log: syslog appender address is empty")
	}
	if strings.TrimSpace(appender.facilityName) != "" {
		facility, ok := parseFacility(appender.facilityName)
		if !ok {
			return nil, fmt.Errorf("goark-log: syslog appender facility %q is unsupported", appender.facilityName)
		}
		appender.facility = facility
	}
	appender.appName = syslogNilValue(appender.appName)
	appender.hostname = syslogNilValue(appender.hostname)
	if appender.layout == nil {
		appender.layout = goarklog.TextLayout{}
	}
	return appender, nil
}

func (a *Appender) Name() string {
	if a == nil || a.name == "" {
		return "syslog"
	}
	return a.name
}

func (a *Appender) Append(ctx context.Context, event goarklog.Event) error {
	if a == nil {
		return fmt.Errorf("goark-log: syslog appender is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var msg bytes.Buffer
	if err := a.layout.Format(&msg, event); err != nil {
		return err
	}
	var packet bytes.Buffer
	packet.WriteByte('<')
	packet.WriteString(fmt.Sprint(a.facility*8 + severity(event.Level)))
	packet.WriteString(">1 ")
	packet.WriteString(eventTime(event.Time).Format(time.RFC3339Nano))
	packet.WriteByte(' ')
	packet.WriteString(a.hostname)
	packet.WriteByte(' ')
	packet.WriteString(a.appName)
	packet.WriteString(" - - - ")
	packet.Write(bytes.TrimSpace(msg.Bytes()))
	packet.WriteByte('\n')

	a.mu.Lock()
	defer a.mu.Unlock()
	conn, err := a.connection()
	if err != nil {
		return err
	}
	_ = conn.SetWriteDeadline(time.Now().Add(a.writeTimeout))
	if _, err := conn.Write(packet.Bytes()); err != nil {
		a.closeLocked()
		return fmt.Errorf("goark-log: syslog appender write: %w", err)
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
		return nil, fmt.Errorf("goark-log: syslog appender dial %s %s: %w", a.network, a.address, err)
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

func severity(level slog.Level) int {
	switch {
	case level >= slog.LevelError:
		return 3
	case level >= slog.LevelWarn:
		return 4
	case level >= slog.LevelInfo:
		return 6
	default:
		return 7
	}
}

func eventTime(when time.Time) time.Time {
	if when.IsZero() {
		return time.Now()
	}
	return when
}

func syslogNilValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return strings.Map(func(r rune) rune {
		if r == ' ' || r == ']' || r == '[' {
			return '-'
		}
		return r
	}, value)
}

func parseFacility(value string) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "local0":
		return 16, true
	case "local1":
		return 17, true
	case "local2":
		return 18, true
	case "local3":
		return 19, true
	case "local4":
		return 20, true
	case "local5":
		return 21, true
	case "local6":
		return 22, true
	case "local7":
		return 23, true
	default:
		return 0, false
	}
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
