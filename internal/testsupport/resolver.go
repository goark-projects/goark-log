package testsupport

import (
	"bytes"

	"goark.dev/log/internal/logvalue"
)

// ConstantJSONResolver 输出固定 JSON 字符串。
type ConstantJSONResolver string

// AppendJSON 写入固定值。
func (r ConstantJSONResolver) AppendJSON(buf *bytes.Buffer, _ Event) {
	logvalue.AppendJSONString(buf, string(r))
}
