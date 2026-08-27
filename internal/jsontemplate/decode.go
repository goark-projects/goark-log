package jsontemplate

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// RawField 保存模板字段名和未解析的 resolver 配置。
type RawField struct {
	Key string
	Raw json.RawMessage
}

// DecodeRawFields 解码 JSON Template 顶层字段，保持字段顺序。
func DecodeRawFields(template string) ([]RawField, error) {
	decoder := json.NewDecoder(strings.NewReader(template))
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '{' {
		return nil, fmt.Errorf("event template must be a JSON object")
	}
	fields := make([]RawField, 0, 8)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := token.(string)
		if !ok {
			return nil, fmt.Errorf("event template field key must be a string")
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return nil, err
		}
		fields = append(fields, RawField{Key: key, Raw: append([]byte(nil), raw...)})
	}
	token, err = decoder.Token()
	if err != nil {
		return nil, err
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '}' {
		return nil, fmt.Errorf("event template object is not closed")
	}
	if token, err = decoder.Token(); err != io.EOF {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("event template has trailing token %v", token)
	}
	return fields, nil
}
