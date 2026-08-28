//go:build !go1.27 && (amd64 || arm64)

// Package jsoncodec 封装核心库的 JSON 回退编码实现。
package jsoncodec

import "github.com/bytedance/sonic"

// Marshal 使用 Sonic 对受支持工具链与架构上的复杂 Go 值执行 JSON 编码。
func Marshal(value any) ([]byte, error) {
	return sonic.ConfigFastest.Marshal(value)
}

// Unmarshal 使用 Sonic 解析受支持工具链与架构上的 JSON 配置片段。
func Unmarshal(data []byte, value any) error {
	return sonic.ConfigFastest.Unmarshal(data, value)
}
