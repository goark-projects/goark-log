// Package jsoncodec 封装核心库的 JSON 回退编码实现。
package jsoncodec

import "github.com/bytedance/sonic"

// Marshal 使用 Sonic 对复杂 Go 值执行 JSON 编码。
func Marshal(value any) ([]byte, error) {
	return sonic.ConfigFastest.Marshal(value)
}
