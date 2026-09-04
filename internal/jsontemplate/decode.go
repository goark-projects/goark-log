package jsontemplate

import (
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
)

// RawField 保存模板字段名和未解析的 resolver 配置。
type RawField struct {
	Key string
	Raw sonic.NoCopyRawMessage
}

// DecodeRawFields 解码 JSON Template 顶层字段，保持字段顺序。
func DecodeRawFields(template string) ([]RawField, error) {
	if !sonic.ValidString(template) {
		return nil, fmt.Errorf("event template is invalid JSON")
	}
	root, err := sonic.GetFromString(template)
	if err != nil {
		return nil, err
	}
	if root.Type() != ast.V_OBJECT {
		return nil, fmt.Errorf("event template must be a JSON object")
	}
	if err := root.Load(); err != nil {
		return nil, err
	}
	count, err := root.Len()
	if err != nil {
		return nil, err
	}
	fields := make([]RawField, 0, count)
	for index := 0; index < count; index++ {
		pair := root.IndexPair(index)
		if pair == nil {
			return nil, fmt.Errorf("event template field %d is unavailable", index)
		}
		raw, err := pair.Value.Raw()
		if err != nil {
			return nil, err
		}
		fields = append(fields, RawField{Key: pair.Key, Raw: sonic.NoCopyRawMessage(append([]byte(nil), raw...))})
	}
	return fields, nil
}
