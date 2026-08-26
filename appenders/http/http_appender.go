package httpappender

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	goarklog "goark.dev/log"
)

const defaultTimeout = 5 * time.Second

// Appender 把日志事件按 HTTP 请求发送到外部系统。
type Appender struct {
	name    string
	url     string
	method  string
	layout  goarklog.Layout
	client  *http.Client
	headers http.Header
}

// Option 调整 HTTP appender。
type Option func(*Appender)

// WithName 设置 appender 名称。
func WithName(name string) Option {
	return func(appender *Appender) {
		appender.name = name
	}
}

// WithMethod 设置 HTTP 方法。
func WithMethod(method string) Option {
	return func(appender *Appender) {
		appender.method = method
	}
}

// WithLayout 设置日志布局。
func WithLayout(layout goarklog.Layout) Option {
	return func(appender *Appender) {
		appender.layout = layout
	}
}

// WithClient 设置 HTTP 客户端。
func WithClient(client *http.Client) Option {
	return func(appender *Appender) {
		appender.client = client
	}
}

// WithHeader 设置固定 HTTP 头。
func WithHeader(key string, value string) Option {
	return func(appender *Appender) {
		key = strings.TrimSpace(key)
		if key == "" {
			return
		}
		appender.headers.Set(key, value)
	}
}

// Register 把 HTTP appender 注册到指定插件表。
func Register(registry *goarklog.PluginRegistry) error {
	if registry == nil {
		registry = goarklog.DefaultPluginRegistry()
	}
	return registry.RegisterAppender("http", func(config goarklog.AppenderBuildConfig) (goarklog.Appender, error) {
		options := []Option{
			WithName(config.Name),
			WithMethod(config.Method),
			WithLayout(config.Layout),
		}
		if timeout, err := parseDuration(config.WriteTimeout); err != nil {
			return nil, err
		} else if timeout > 0 {
			options = append(options, WithClient(&http.Client{Timeout: timeout}))
		}
		return New(firstNonBlank(config.URL, config.Target), options...)
	})
}

// New 创建 HTTP appender。
func New(url string, options ...Option) (*Appender, error) {
	appender := &Appender{
		name:    "http",
		url:     strings.TrimSpace(url),
		method:  http.MethodPost,
		layout:  goarklog.JSONLayout{},
		client:  &http.Client{Timeout: defaultTimeout},
		headers: make(http.Header),
	}
	for _, option := range options {
		if option != nil {
			option(appender)
		}
	}
	if strings.TrimSpace(appender.name) == "" {
		return nil, fmt.Errorf("goark-log: http appender name is empty")
	}
	if appender.url == "" {
		return nil, fmt.Errorf("goark-log: http appender url is empty")
	}
	if strings.TrimSpace(appender.method) == "" {
		appender.method = http.MethodPost
	}
	appender.method = strings.ToUpper(strings.TrimSpace(appender.method))
	if appender.layout == nil {
		appender.layout = goarklog.JSONLayout{}
	}
	if appender.client == nil {
		appender.client = &http.Client{Timeout: defaultTimeout}
	}
	if appender.headers.Get("Content-Type") == "" {
		appender.headers.Set("Content-Type", "application/json")
	}
	return appender, nil
}

func (a *Appender) Name() string {
	if a == nil || a.name == "" {
		return "http"
	}
	return a.name
}

func (a *Appender) Append(ctx context.Context, event goarklog.Event) error {
	if a == nil {
		return fmt.Errorf("goark-log: http appender is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var buf bytes.Buffer
	if err := a.layout.Format(&buf, event); err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, a.method, a.url, bytes.NewReader(buf.Bytes()))
	if err != nil {
		return fmt.Errorf("goark-log: create http appender request: %w", err)
	}
	request.Header = a.headers.Clone()
	response, err := a.client.Do(request)
	if err != nil {
		return fmt.Errorf("goark-log: send http appender request: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("goark-log: http appender status %d", response.StatusCode)
	}
	return nil
}

func (a *Appender) Close() error {
	return nil
}

func parseDuration(value string) (time.Duration, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	duration, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("goark-log: invalid http appender timeout %q", value)
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
