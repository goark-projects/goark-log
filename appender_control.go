package goarklog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"strings"
	"sync"

	internaldelegate "goark.dev/log/internal/delegating"
	internalfileappender "goark.dev/log/internal/fileappender"
	internaljsonappender "goark.dev/log/internal/jsonappender"
)

// Appender 是日志事件的最终写出端。
type Appender interface {
	Name() string
	Append(ctx context.Context, event Event) error
	Close() error
}

var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

const (
	// DefaultFileBufferSize 是文件 appender 默认缓冲大小。
	DefaultFileBufferSize = internalfileappender.DefaultFileBufferSize
)

// AppenderRef 描述一次到 appender 的结构化引用。
type AppenderRef struct {
	Ref             string
	Level           *slog.Level
	IncludeLocation *bool
	Filters         []Filter
}

// AppenderRefOption 调整结构化 appender 引用。
type AppenderRefOption func(*AppenderRef)

// NewAppenderRef 创建结构化 appender 引用。
func NewAppenderRef(ref string, options ...AppenderRefOption) AppenderRef {
	config := AppenderRef{Ref: ref}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	return config
}

// WithAppenderRefLevel 设置当前引用独有的级别下限。
func WithAppenderRefLevel(level slog.Level) AppenderRefOption {
	return func(config *AppenderRef) {
		copied := level
		config.Level = &copied
	}
}

// WithAppenderRefLocation 设置当前引用是否采集调用位置。
func WithAppenderRefLocation(enabled bool) AppenderRefOption {
	return func(config *AppenderRef) {
		copied := enabled
		config.IncludeLocation = &copied
	}
}

// WithAppenderRefFilters 设置当前引用独有的过滤器链。
func WithAppenderRefFilters(filters ...Filter) AppenderRefOption {
	return func(config *AppenderRef) {
		config.Filters = append(config.Filters, filters...)
	}
}

type appenderControl struct {
	ref             string
	level           *slog.Level
	includeLocation *bool
	filters         []Filter
	appender        Appender
}

type controlledAppender struct {
	control appenderControl
}

// ConsoleAppender 把日志写入 stdout、stderr 或自定义 writer。
type ConsoleAppender = internalfileappender.ConsoleAppender

// ConsoleOption 调整 ConsoleAppender。
type ConsoleOption = internalfileappender.ConsoleOption

// FileAppender 把日志追加写入普通文件。
type FileAppender = internalfileappender.FileAppender

// FileOption 调整 FileAppender。
type FileOption = internalfileappender.FileOption

// WithConsoleName 设置 appender 名称。
func WithConsoleName(name string) ConsoleOption {
	return internalfileappender.WithConsoleName(name)
}

// WithConsoleWriter 设置输出 writer，主要用于测试和嵌入式场景。
func WithConsoleWriter(writer io.Writer) ConsoleOption {
	return internalfileappender.WithConsoleWriter(writer)
}

// WithConsoleLayout 设置日志布局。
func WithConsoleLayout(layout Layout) ConsoleOption {
	return internalfileappender.WithConsoleLayout(layout)
}

// NewConsoleAppender 创建控制台 appender。
func NewConsoleAppender(options ...ConsoleOption) *ConsoleAppender {
	return internalfileappender.NewConsoleAppender(options...)
}

// WithFileName 设置 appender 名称。
func WithFileName(name string) FileOption {
	return internalfileappender.WithFileName(name)
}

// WithFileLayout 设置日志布局。
func WithFileLayout(layout Layout) FileOption {
	return internalfileappender.WithFileLayout(layout)
}

// WithFileBufferSize 设置文件写缓冲大小，0 表示禁用缓冲。
func WithFileBufferSize(size int) FileOption {
	return internalfileappender.WithFileBufferSize(size)
}

// WithFileFlushOnWrite 设置每次写入后立即 flush。
func WithFileFlushOnWrite(enabled bool) FileOption {
	return internalfileappender.WithFileFlushOnWrite(enabled)
}

// WithFileAppend 设置打开文件时是否追加到已有内容。
func WithFileAppend(enabled bool) FileOption {
	return internalfileappender.WithFileAppend(enabled)
}

// WithFileCreateOnDemand 设置是否延迟到首次写入时创建文件。
func WithFileCreateOnDemand(enabled bool) FileOption {
	return internalfileappender.WithFileCreateOnDemand(enabled)
}

// WithFilePermissions 设置新建日志文件权限。
func WithFilePermissions(permissions fs.FileMode) FileOption {
	return internalfileappender.WithFilePermissions(permissions)
}

// NewFileAppender 创建普通文件 appender。
func NewFileAppender(path string, options ...FileOption) (*FileAppender, error) {
	return internalfileappender.NewFileAppender(path, options...)
}

// JSONAppender 将事件直接编码为单行 JSON，适合极低分配热路径。
type JSONAppender = internaljsonappender.Appender

// JSONAppenderOption 调整 JSONAppender。
type JSONAppenderOption = internaljsonappender.Option

// WithJSONAppenderName 设置 appender 名称。
func WithJSONAppenderName(name string) JSONAppenderOption {
	return internaljsonappender.WithName(name)
}

// WithJSONAppenderWriter 设置输出 writer，主要用于测试、基准和嵌入式直写场景。
func WithJSONAppenderWriter(writer io.Writer) JSONAppenderOption {
	return internaljsonappender.WithWriter(writer)
}

// WithJSONAppenderBufferSize 设置文件输出缓冲大小，0 表示禁用应用层缓冲。
func WithJSONAppenderBufferSize(size int) JSONAppenderOption {
	return internaljsonappender.WithBufferSize(size)
}

// WithJSONAppenderFlushOnWrite 设置每次写入后立即刷新应用层缓冲。
func WithJSONAppenderFlushOnWrite(enabled bool) JSONAppenderOption {
	return internaljsonappender.WithFlushOnWrite(enabled)
}

// NewJSONAppender 创建 JSON 直写 appender。
func NewJSONAppender(options ...JSONAppenderOption) *JSONAppender {
	return internaljsonappender.New(options...)
}

// NewJSONFileAppender 创建面向文件的 JSON 直写 appender。
func NewJSONFileAppender(path string, options ...JSONAppenderOption) (*JSONAppender, error) {
	return internaljsonappender.NewFile(path, options...)
}

// FailoverAppender 在主 appender 写入失败时按顺序尝试备用 appender。
type FailoverAppender = internaldelegate.FailoverAppender

// FailoverOption 调整 FailoverAppender。
type FailoverOption = internaldelegate.FailoverOption

// RouteKeyFunc 从事件中计算路由键。
type RouteKeyFunc = internaldelegate.RouteKeyFunc

// RoutingAppender 按事件属性或自定义函数选择下游 appender。
type RoutingAppender = internaldelegate.RoutingAppender

// RoutingOption 调整 RoutingAppender。
type RoutingOption = internaldelegate.RoutingOption

// RewritePolicy 在写出前重写事件快照。
type RewritePolicy = internaldelegate.RewritePolicy

// RewriteAppender 在写出前执行事件重写。
type RewriteAppender = internaldelegate.RewriteAppender

// RewriteOption 调整 RewriteAppender。
type RewriteOption = internaldelegate.RewriteOption

// WithFailoverName 设置 failover appender 名称。
func WithFailoverName(name string) FailoverOption {
	return internaldelegate.WithFailoverName(name)
}

// WithFailoverCloseChildren 设置关闭 failover 时是否关闭下游 appender。
func WithFailoverCloseChildren(enabled bool) FailoverOption {
	return internaldelegate.WithFailoverCloseChildren(enabled)
}

// NewFailoverAppender 创建失败转移 appender。
func NewFailoverAppender(primary Appender, failovers []Appender, options ...FailoverOption) (*FailoverAppender, error) {
	return internaldelegate.NewFailoverAppender(primary, delegatingAppenders(failovers), options...)
}

// WithRoutingName 设置 routing appender 名称。
func WithRoutingName(name string) RoutingOption {
	return internaldelegate.WithRoutingName(name)
}

// WithRoutingAttrKey 设置按事件属性取路由键。
func WithRoutingAttrKey(key string) RoutingOption {
	return internaldelegate.WithRoutingAttrKey(key)
}

// WithRoutingDefault 设置未命中路由时的默认 appender。
func WithRoutingDefault(route Appender) RoutingOption {
	return internaldelegate.WithRoutingDefault(route)
}

// WithRoutingKeyFunc 设置自定义路由键函数。
func WithRoutingKeyFunc(routeKeyFunc RouteKeyFunc) RoutingOption {
	return internaldelegate.WithRoutingKeyFunc(routeKeyFunc)
}

// WithRoutingCloseChildren 设置关闭 routing 时是否关闭下游 appender。
func WithRoutingCloseChildren(enabled bool) RoutingOption {
	return internaldelegate.WithRoutingCloseChildren(enabled)
}

// NewRoutingAppender 创建路由 appender。
func NewRoutingAppender(routes map[string]Appender, options ...RoutingOption) (*RoutingAppender, error) {
	return internaldelegate.NewRoutingAppender(delegatingAppenderMap(routes), options...)
}

// WithRewriteName 设置 rewrite appender 名称。
func WithRewriteName(name string) RewriteOption {
	return internaldelegate.WithRewriteName(name)
}

// WithRewriteCloseDelegate 设置关闭 rewrite 时是否关闭下游 appender。
func WithRewriteCloseDelegate(enabled bool) RewriteOption {
	return internaldelegate.WithRewriteCloseDelegate(enabled)
}

// NewRewriteAppender 创建事件重写 appender。
func NewRewriteAppender(delegate Appender, policy RewritePolicy, options ...RewriteOption) (*RewriteAppender, error) {
	return internaldelegate.NewRewriteAppender(delegate, policy, options...)
}

func delegatingAppenders(appenders []Appender) []internaldelegate.Appender {
	if len(appenders) == 0 {
		return nil
	}
	converted := make([]internaldelegate.Appender, 0, len(appenders))
	for _, appender := range appenders {
		converted = append(converted, appender)
	}
	return converted
}

func delegatingAppenderMap(routes map[string]Appender) map[string]internaldelegate.Appender {
	if len(routes) == 0 {
		return nil
	}
	converted := make(map[string]internaldelegate.Appender, len(routes))
	for key, appender := range routes {
		converted[key] = appender
	}
	return converted
}

// FilteredAppender 为任意 appender 增加过滤器链。
type FilteredAppender struct {
	delegate Appender
	filters  []Filter
}

// NewFilteredAppender 创建带过滤器链的 appender。
func NewFilteredAppender(delegate Appender, filters ...Filter) (*FilteredAppender, error) {
	if delegate == nil {
		return nil, fmt.Errorf("goark-log: filtered appender delegate is nil")
	}
	chain, err := normalizeFilters("appender "+delegate.Name(), filters)
	if err != nil {
		return nil, err
	}
	return &FilteredAppender{delegate: delegate, filters: chain}, nil
}

func newAppenderControl(appenderByName map[string]Appender, ref AppenderRef) (appenderControl, error) {
	name := strings.TrimSpace(ref.Ref)
	if name == "" {
		return appenderControl{}, fmt.Errorf("appender ref is empty")
	}
	appender, ok := appenderByName[name]
	if !ok {
		return appenderControl{}, fmt.Errorf("appender %q is not configured", name)
	}
	filters, err := normalizeFilters("appender ref "+name, ref.Filters)
	if err != nil {
		return appenderControl{}, err
	}
	control := appenderControl{
		ref:      name,
		filters:  filters,
		appender: appender,
	}
	if ref.Level != nil {
		level := *ref.Level
		control.level = &level
	}
	if ref.IncludeLocation != nil {
		includeLocation := *ref.IncludeLocation
		control.includeLocation = &includeLocation
	}
	return control, nil
}

func (c appenderControl) Append(ctx context.Context, event Event) error {
	_, err := c.append(ctx, event)
	return err
}

// append 返回底层 appender 是否被实际调用，供核心指标区分跳过和写入。
func (c appenderControl) append(ctx context.Context, event Event) (bool, error) {
	if c.appender == nil {
		return false, nil
	}
	if c.level != nil && event.Level < *c.level {
		return false, nil
	}
	if applyFilters(ctx, c.filters, event) == FilterDeny {
		return false, nil
	}
	if c.includeLocation != nil && !*c.includeLocation {
		event.PC = 0
	}
	return true, c.appender.Append(ctx, event)
}

func (c appenderControl) requiresLocation() bool {
	return c.includeLocation != nil && *c.includeLocation
}

func (c appenderControl) name() string {
	if c.appender == nil {
		return c.ref
	}
	return c.appender.Name()
}

func (a controlledAppender) Name() string {
	return a.control.name()
}

func (a controlledAppender) Append(ctx context.Context, event Event) error {
	_, err := a.control.append(ctx, event)
	return err
}

func (a controlledAppender) Close() error {
	if a.control.appender == nil {
		return nil
	}
	return a.control.appender.Close()
}

func resolveAppenderControls(appenderByName map[string]Appender, refs []AppenderRef) ([]appenderControl, error) {
	controls := make([]appenderControl, 0, len(refs))
	for _, ref := range refs {
		control, err := newAppenderControl(appenderByName, ref)
		if err != nil {
			return nil, err
		}
		controls = append(controls, control)
	}
	return controls, nil
}

func appendUniqueAppenderControls(dst []appenderControl, src []appenderControl) []appenderControl {
	seen := make(map[string]struct{}, len(dst)+len(src))
	out := dst[:0]
	for _, control := range dst {
		name := control.name()
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, control)
	}
	for _, control := range src {
		name := control.name()
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, control)
	}
	return out
}

func (a *FilteredAppender) Name() string {
	if a == nil || a.delegate == nil {
		return ""
	}
	return a.delegate.Name()
}

func (a *FilteredAppender) Append(ctx context.Context, event Event) error {
	if a == nil || a.delegate == nil {
		return fmt.Errorf("goark-log: filtered appender is nil")
	}
	ctx = normalizeContext(ctx)
	if applyFilters(ctx, a.filters, event) == FilterDeny {
		return nil
	}
	return a.delegate.Append(ctx, event)
}

func (a *FilteredAppender) Close() error {
	if a == nil || a.delegate == nil {
		return nil
	}
	return a.delegate.Close()
}

func isAsyncAppender(appender Appender) bool {
	switch value := appender.(type) {
	case *AsyncAppender:
		return true
	case *FilteredAppender:
		return isAsyncAppender(value.delegate)
	default:
		return false
	}
}

func mergeAppenderRefs(simple []string, controls []AppenderRef) []AppenderRef {
	if len(simple) == 0 && len(controls) == 0 {
		return nil
	}
	refs := make([]AppenderRef, 0, len(simple)+len(controls))
	for _, ref := range simple {
		refs = append(refs, AppenderRef{Ref: ref})
	}
	for _, ref := range controls {
		refs = append(refs, copyAppenderRef(ref))
	}
	return refs
}

func copyAppenderRef(ref AppenderRef) AppenderRef {
	copied := AppenderRef{
		Ref:     ref.Ref,
		Filters: append([]Filter(nil), ref.Filters...),
	}
	if ref.Level != nil {
		level := *ref.Level
		copied.Level = &level
	}
	if ref.IncludeLocation != nil {
		includeLocation := *ref.IncludeLocation
		copied.IncludeLocation = &includeLocation
	}
	return copied
}

func releaseBuffer(buf *bytes.Buffer) {
	if buf.Cap() > 64*1024 {
		return
	}
	buf.Reset()
	bufferPool.Put(buf)
}
