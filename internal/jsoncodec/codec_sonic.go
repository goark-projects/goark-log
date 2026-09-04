// Package jsoncodec 封装日志框架唯一的 JSON 编解码实现。
package jsoncodec

import "github.com/bytedance/sonic"

// Marshal 使用 Sonic 对复杂 Go 值执行 JSON 编码。
func Marshal(value any) ([]byte, error) {
	return sonic.ConfigFastest.Marshal(value)
}

// Unmarshal 使用 Sonic 解析 JSON 配置片段。
func Unmarshal(data []byte, value any) error {
	return sonic.ConfigFastest.Unmarshal(data, value)
}
