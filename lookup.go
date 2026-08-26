package goarklog

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// LookupFunc 根据键解析配置变量。
type LookupFunc func(key string) (string, bool)

// LookupResolver 负责解析配置中的 ${namespace:key} 变量。
type LookupResolver struct {
	lookups map[string]LookupFunc
	now     func() time.Time
}

// NewLookupResolver 创建带默认 lookup 的解析器。
func NewLookupResolver() *LookupResolver {
	resolver := &LookupResolver{
		lookups: make(map[string]LookupFunc, 4),
		now:     time.Now,
	}
	resolver.Register("env", os.LookupEnv)
	resolver.Register("sys", resolver.systemLookup)
	resolver.Register("go", resolver.goLookup)
	resolver.Register("date", resolver.dateLookup)
	return resolver
}

// Register 注册一个命名 lookup。
func (r *LookupResolver) Register(namespace string, lookup LookupFunc) {
	if r == nil || lookup == nil {
		return
	}
	namespace = strings.ToLower(strings.TrimSpace(namespace))
	if namespace == "" {
		return
	}
	if isBlockedLookupNamespace(namespace) {
		return
	}
	r.lookups[namespace] = lookup
}

func (r *LookupResolver) clone() *LookupResolver {
	if r == nil {
		return NewLookupResolver()
	}
	copied := &LookupResolver{
		lookups: make(map[string]LookupFunc, len(r.lookups)),
		now:     r.now,
	}
	for namespace, lookup := range r.lookups {
		copied.lookups[namespace] = lookup
	}
	return copied
}

// Resolve 替换文本中的配置变量。
func (r *LookupResolver) Resolve(text string) (string, error) {
	if !strings.Contains(text, "${") {
		return text, nil
	}
	if r == nil {
		r = NewLookupResolver()
	}
	var builder strings.Builder
	builder.Grow(len(text))
	for index := 0; index < len(text); {
		if text[index] == '$' && index+1 < len(text) && text[index+1] == '$' {
			builder.WriteByte('$')
			index += 2
			continue
		}
		if text[index] != '$' || index+1 >= len(text) || text[index+1] != '{' {
			builder.WriteByte(text[index])
			index++
			continue
		}
		end := strings.IndexByte(text[index+2:], '}')
		if end < 0 {
			return "", fmt.Errorf("goark-log: lookup expression is not closed in %q", text)
		}
		expr := text[index+2 : index+2+end]
		value, err := r.resolveExpr(expr)
		if err != nil {
			return "", err
		}
		builder.WriteString(value)
		index += end + 3
	}
	return builder.String(), nil
}

func (r *LookupResolver) resolveExpr(expr string) (string, error) {
	if key, fallback, hasFallback, ok, err := splitPropertyShorthandExpr(expr); ok || err != nil {
		if err != nil {
			return "", err
		}
		if value, ok := r.propertyLookup(key); ok {
			return value, nil
		}
		if hasFallback {
			return fallback, nil
		}
		return "", fmt.Errorf("goark-log: property lookup %q has no value", key)
	}
	namespace, key, fallback, hasFallback, err := splitLookupExpr(expr)
	if err != nil {
		return "", err
	}
	lookup, ok := r.lookups[namespace]
	if !ok {
		return "", fmt.Errorf("goark-log: lookup namespace %q is not registered", namespace)
	}
	if value, ok := lookup(key); ok {
		return value, nil
	}
	if hasFallback {
		return fallback, nil
	}
	return "", fmt.Errorf("goark-log: lookup %q has no value", expr)
}

func (r *LookupResolver) propertyLookup(key string) (string, bool) {
	for _, namespace := range []string{"prop", "property"} {
		lookup, ok := r.lookups[namespace]
		if !ok {
			continue
		}
		if value, ok := lookup(key); ok {
			return value, true
		}
	}
	return "", false
}

func splitPropertyShorthandExpr(expr string) (key string, fallback string, hasFallback bool, ok bool, err error) {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return "", "", false, true, fmt.Errorf("goark-log: lookup expression is empty")
	}
	key = trimmed
	if before, after, hasDefault := strings.Cut(trimmed, ":-"); hasDefault {
		key = strings.TrimSpace(before)
		fallback = after
		hasFallback = true
	}
	if strings.Contains(key, ":") {
		return "", "", false, false, nil
	}
	if key == "" {
		return "", "", false, true, fmt.Errorf("goark-log: lookup expression %q key is empty", expr)
	}
	return key, fallback, hasFallback, true, nil
}

func splitLookupExpr(expr string) (namespace string, key string, fallback string, hasFallback bool, err error) {
	if strings.TrimSpace(expr) == "" {
		err = fmt.Errorf("goark-log: lookup expression is empty")
		return
	}
	head, tail, ok := strings.Cut(expr, ":")
	if !ok {
		err = fmt.Errorf("goark-log: lookup expression %q must use namespace:key", expr)
		return
	}
	namespace = strings.ToLower(strings.TrimSpace(head))
	key = strings.TrimSpace(tail)
	if namespace == "" || key == "" {
		err = fmt.Errorf("goark-log: lookup expression %q must use namespace:key", expr)
		return
	}
	if before, after, ok := strings.Cut(key, ":-"); ok {
		key = strings.TrimSpace(before)
		fallback = after
		hasFallback = true
	}
	if key == "" {
		err = fmt.Errorf("goark-log: lookup expression %q key is empty", expr)
	}
	return
}

func (r *LookupResolver) systemLookup(key string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "pid", "processid", "process-id":
		return strconv.Itoa(os.Getpid()), true
	case "hostname", "host":
		hostname, err := os.Hostname()
		return hostname, err == nil
	case "cwd", "workdir", "workingdir", "working-dir":
		wd, err := os.Getwd()
		return wd, err == nil
	case "os":
		return runtime.GOOS, true
	case "arch":
		return runtime.GOARCH, true
	default:
		return "", false
	}
}

func (r *LookupResolver) goLookup(key string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "version":
		return runtime.Version(), true
	case "os":
		return runtime.GOOS, true
	case "arch":
		return runtime.GOARCH, true
	default:
		return "", false
	}
}

func (r *LookupResolver) dateLookup(key string) (string, bool) {
	if strings.TrimSpace(key) == "" {
		return "", false
	}
	now := time.Now
	if r.now != nil {
		now = r.now
	}
	layout, unixMode := normalizeTimePattern(key)
	when := now()
	switch unixMode {
	case timeUnixSeconds:
		return strconv.FormatInt(when.Unix(), 10), true
	case timeUnixMillis:
		return strconv.FormatInt(when.UnixMilli(), 10), true
	case timeUnixMicros:
		return strconv.FormatInt(when.UnixMicro(), 10), true
	case timeUnixNanos:
		return strconv.FormatInt(when.UnixNano(), 10), true
	default:
		return when.Format(layout), true
	}
}
