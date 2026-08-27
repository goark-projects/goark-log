package configxml

import (
	"fmt"
	"strconv"
	"strings"
)

// Int 解析 XML 属性中的整数，空值返回零。
func Int(value string, field string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("%s is invalid", field)
	}
	return parsed, nil
}

// IntPointer 解析可选整数，空值或非法值返回 nil。
func IntPointer(value string) *int {
	parsed, err := Int(value, "")
	if err != nil || parsed == 0 {
		return nil
	}
	return &parsed
}

// IntValue 解析整数，非法值按零值处理。
func IntValue(value string) int {
	parsed, err := Int(value, "")
	if err != nil {
		return 0
	}
	return parsed
}

// Bool 解析 XML 属性中的布尔值，空值返回 false。
func Bool(value string, field string) (bool, error) {
	if strings.TrimSpace(value) == "" {
		return false, nil
	}
	parsed, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(value)))
	if err != nil {
		return false, fmt.Errorf("%s is invalid", field)
	}
	return parsed, nil
}

// BoolPointer 解析可选布尔值，空值或非法值返回 nil。
func BoolPointer(value string) *bool {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parsed, err := Bool(value, "")
	if err != nil {
		return nil
	}
	return &parsed
}

// BoolPointerStrict 解析可选布尔值，非法值返回错误。
func BoolPointerStrict(value string, field string) (*bool, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := Bool(value, field)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
