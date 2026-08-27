package configvalue

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// ByteSize 解析日志配置中的字节大小。
func ByteSize(value string) (int64, error) {
	text := strings.TrimSpace(value)
	if text == "" {
		return 0, fmt.Errorf("goark-log: byte size is empty")
	}
	if text == "0" {
		return 0, nil
	}
	index := 0
	for index < len(text) {
		r := rune(text[index])
		if !(unicode.IsDigit(r) || r == '.') {
			break
		}
		index++
	}
	numberText := strings.TrimSpace(text[:index])
	unitText := strings.ToLower(strings.TrimSpace(text[index:]))
	if numberText == "" {
		return 0, fmt.Errorf("goark-log: invalid byte size %q", value)
	}
	number, err := strconv.ParseFloat(numberText, 64)
	if err != nil {
		return 0, fmt.Errorf("goark-log: invalid byte size %q", value)
	}
	if number < 0 {
		return 0, fmt.Errorf("goark-log: byte size must be >= 0")
	}
	multiplier, ok := byteSizeMultiplier(unitText)
	if !ok {
		return 0, fmt.Errorf("goark-log: unsupported byte size unit %q", unitText)
	}
	if number >= float64(maxInt64)/float64(multiplier) {
		return 0, fmt.Errorf("goark-log: byte size overflow %q", value)
	}
	size := int64(number * float64(multiplier))
	return size, nil
}

// RollingInterval 解析时间滚动间隔。
func RollingInterval(value string) (time.Duration, error) {
	text := strings.ToLower(strings.TrimSpace(value))
	switch text {
	case "", "0", "off", "none", "disabled":
		return 0, nil
	case "minute", "minutely":
		return time.Minute, nil
	case "hour", "hourly":
		return time.Hour, nil
	case "day", "daily":
		return 24 * time.Hour, nil
	default:
		if interval, ok, err := rollingIntervalWithDayUnit(text); ok || err != nil {
			return interval, err
		}
		interval, err := time.ParseDuration(text)
		if err != nil {
			return 0, fmt.Errorf("goark-log: invalid rolling interval %q", value)
		}
		if interval < 0 {
			return 0, fmt.Errorf("goark-log: rolling interval must be >= 0")
		}
		return interval, nil
	}
}

func rollingIntervalWithDayUnit(text string) (time.Duration, bool, error) {
	units := []struct {
		suffix string
		scale  time.Duration
	}{
		{suffix: "minutes", scale: time.Minute},
		{suffix: "minute", scale: time.Minute},
		{suffix: "mins", scale: time.Minute},
		{suffix: "min", scale: time.Minute},
		{suffix: "hours", scale: time.Hour},
		{suffix: "hour", scale: time.Hour},
		{suffix: "days", scale: 24 * time.Hour},
		{suffix: "day", scale: 24 * time.Hour},
	}
	for _, unit := range units {
		if !strings.HasSuffix(text, unit.suffix) {
			continue
		}
		numberText := strings.TrimSpace(strings.TrimSuffix(text, unit.suffix))
		if numberText == "" {
			return 0, true, fmt.Errorf("goark-log: invalid rolling interval %q", text)
		}
		number, err := strconv.ParseFloat(numberText, 64)
		if err != nil || number < 0 {
			return 0, true, fmt.Errorf("goark-log: invalid rolling interval %q", text)
		}
		return time.Duration(number * float64(unit.scale)), true, nil
	}
	return 0, false, nil
}

// RollingMaxAge 解析滚动档案最大保留时间。
func RollingMaxAge(value string) (time.Duration, error) {
	text := strings.ToLower(strings.TrimSpace(value))
	switch text {
	case "", "0", "off", "none", "disabled":
		return 0, nil
	}
	if strings.HasSuffix(text, "d") {
		days, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(text, "d")), 64)
		if err != nil || days < 0 {
			return 0, fmt.Errorf("goark-log: invalid rolling max age %q", value)
		}
		return time.Duration(days * float64(24*time.Hour)), nil
	}
	if strings.HasSuffix(text, "day") || strings.HasSuffix(text, "days") {
		number := strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(text, "days"), "day"))
		days, err := strconv.ParseFloat(number, 64)
		if err != nil || days < 0 {
			return 0, fmt.Errorf("goark-log: invalid rolling max age %q", value)
		}
		return time.Duration(days * float64(24*time.Hour)), nil
	}
	age, err := time.ParseDuration(text)
	if err != nil {
		return 0, fmt.Errorf("goark-log: invalid rolling max age %q", value)
	}
	if age < 0 {
		return 0, fmt.Errorf("goark-log: rolling max age must be >= 0")
	}
	return age, nil
}

func byteSizeMultiplier(unit string) (int64, bool) {
	switch unit {
	case "", "b", "byte", "bytes":
		return 1, true
	case "k", "kb":
		return 1000, true
	case "m", "mb":
		return 1000 * 1000, true
	case "g", "gb":
		return 1000 * 1000 * 1000, true
	case "t", "tb":
		return 1000 * 1000 * 1000 * 1000, true
	case "ki", "kib":
		return 1024, true
	case "mi", "mib":
		return 1024 * 1024, true
	case "gi", "gib":
		return 1024 * 1024 * 1024, true
	case "ti", "tib":
		return 1024 * 1024 * 1024 * 1024, true
	default:
		return 0, false
	}
}

const maxInt64 = int64(1<<63 - 1)
