package goarklog

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type jsonTemplateRawField struct {
	key string
	raw json.RawMessage
}

// NewJSONTemplateLayout 编译 JSON 事件模板。

func decodeJSONTemplateRawFields(template string) ([]jsonTemplateRawField, error) {
	decoder := json.NewDecoder(strings.NewReader(template))
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '{' {
		return nil, fmt.Errorf("event template must be a JSON object")
	}
	fields := make([]jsonTemplateRawField, 0, 8)
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
		fields = append(fields, jsonTemplateRawField{key: key, raw: append([]byte(nil), raw...)})
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
