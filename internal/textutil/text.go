package textutil

import (
	"sort"
	"strings"
	"time"
)

// FirstNonBlank 返回首个非空白字符串，返回值已去除首尾空白。
func FirstNonBlank(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// FirstNonZero 返回首个非零整数。
func FirstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

// OptionalDuration 解析可选 duration，空值返回 0，非法值返回 -1。
func OptionalDuration(value string) time.Duration {
	text := strings.TrimSpace(value)
	if text == "" {
		return 0
	}
	duration, err := time.ParseDuration(text)
	if err != nil {
		return -1
	}
	return duration
}

// NormalizeKind 规范化配置和插件类型名。
func NormalizeKind(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, "_", "")
	return value
}

// FirstSlice 返回首个非空切片的快照。
func FirstSlice[S ~[]E, E any](groups ...S) S {
	for _, values := range groups {
		if len(values) == 0 {
			continue
		}
		out := make(S, len(values))
		copy(out, values)
		return out
	}
	var zero S
	return zero
}

// FirstTrimmedStrings 返回首个非空字符串切片的去空白快照。
func FirstTrimmedStrings[S ~[]string](groups ...S) []string {
	for _, values := range groups {
		if len(values) == 0 {
			continue
		}
		out := make([]string, len(values))
		for index, value := range values {
			out[index] = strings.TrimSpace(value)
		}
		return out
	}
	return nil
}

// SortedKeys 返回按字典序排序的 map 键。
func SortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
