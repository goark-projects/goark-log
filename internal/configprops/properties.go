package configprops

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"goark.dev/log/internal/textutil"
)

// Aliases 保存 properties 配置中的 appender/logger 逻辑名称映射。
type Aliases struct {
	appenders map[string]string
	loggers   map[string]string
}

// Read 读取 Java properties 风格的键值配置。
func Read(reader io.Reader) (map[string]string, error) {
	scanner := bufio.NewScanner(reader)
	values := make(map[string]string)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		key, value, ok := cut(line)
		if !ok {
			return nil, fmt.Errorf("goark-log: properties line %d is invalid", lineNumber)
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

// CollectAliases 收集 properties 配置中显式声明的对象名称。
func CollectAliases(values map[string]string) Aliases {
	aliases := Aliases{
		appenders: make(map[string]string),
		loggers:   make(map[string]string),
	}
	for key, value := range values {
		if strings.HasPrefix(key, "appender.") {
			id, field, ok := SplitID(strings.TrimPrefix(key, "appender."))
			if ok && field == "name" && strings.TrimSpace(value) != "" {
				aliases.appenders[id] = strings.TrimSpace(value)
			}
		}
		if strings.HasPrefix(key, "logger.") {
			id, field, ok := SplitID(strings.TrimPrefix(key, "logger."))
			if ok && field == "name" && strings.TrimSpace(value) != "" {
				aliases.loggers[id] = strings.TrimSpace(value)
			}
		}
	}
	return aliases
}

// AppenderName 返回 appender 的有效名称。
func (a Aliases) AppenderName(id string) string {
	if name := strings.TrimSpace(a.appenders[id]); name != "" {
		return name
	}
	return id
}

// LoggerName 返回 logger 的有效名称。
func (a Aliases) LoggerName(id string) string {
	if name := strings.TrimSpace(a.loggers[id]); name != "" {
		return name
	}
	return id
}

// SplitID 拆分 properties 中的 id.field 片段。
func SplitID(key string) (string, string, bool) {
	id, field, ok := strings.Cut(key, ".")
	id = strings.TrimSpace(id)
	field = strings.TrimSpace(field)
	if !ok || id == "" || field == "" {
		return "", "", false
	}
	return id, field, true
}

// SplitFilterPairKey 拆分 filter.<id>.<pair>.key/value 片段。
func SplitFilterPairKey(key string) (string, string, string, bool) {
	if !strings.HasPrefix(key, "filter.") {
		return "", "", "", false
	}
	filterID, field, ok := SplitID(strings.TrimPrefix(key, "filter."))
	if !ok {
		return "", "", "", false
	}
	pairID, pairField, ok := SplitID(field)
	if !ok {
		return "", "", "", false
	}
	normalized := textutil.NormalizeKind(pairID)
	if !strings.HasPrefix(normalized, "keyvaluepair") && !strings.HasPrefix(normalized, "kv") {
		return "", "", "", false
	}
	switch strings.ToLower(pairField) {
	case "key", "value":
		return filterID, pairID, strings.ToLower(pairField), true
	default:
		return "", "", "", false
	}
}

// List 解析逗号或分号分隔的 properties 列表。
func List(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// Int 解析 properties 整数值。
func Int(value string, field string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("goark-log: properties %s value %q is invalid integer", field, value)
	}
	return parsed, nil
}

// Bool 解析 properties 布尔值。
func Bool(value string, field string) (bool, error) {
	parsed, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(value)))
	if err != nil {
		return false, fmt.Errorf("goark-log: properties %s value %q is invalid boolean", field, value)
	}
	return parsed, nil
}

func cut(line string) (string, string, bool) {
	for _, separator := range []string{"=", ":"} {
		key, value, ok := strings.Cut(line, separator)
		if ok {
			return key, value, true
		}
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", "", false
	}
	return fields[0], strings.Join(fields[1:], " "), true
}
