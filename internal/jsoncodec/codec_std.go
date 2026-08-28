//go:build go1.27 || (!amd64 && !arm64)

// Package jsoncodec 封装核心库的 JSON 回退编码实现。
package jsoncodec

import "encoding/json"

// Marshal 使用标准库对复杂 Go 值执行 JSON 编码。
func Marshal(value any) ([]byte, error) {
	return json.Marshal(value)
}

// Unmarshal 使用标准库解析 JSON 配置片段。
func Unmarshal(data []byte, value any) error {
	return json.Unmarshal(data, value)
}
