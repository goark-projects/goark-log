package testsupport

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
)

// WriteConfig 写入规范化的测试配置。
func WriteConfig(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

// ReadTextFile 读取测试文本文件。
func ReadTextFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return string(content)
}

// ReadGzipFile 读取 gzip 测试文件。
func ReadGzipFile(t *testing.T, path string) string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open(%s) error = %v", path, err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("NewReader(%s) error = %v", path, err)
	}
	defer reader.Close()
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll(%s) error = %v", path, err)
	}
	return string(content)
}

// AssertCompleteJSONMessages 校验完整 JSON 日志中的消息顺序。
func AssertCompleteJSONMessages(t *testing.T, path string, want ...string) {
	t.Helper()
	AssertCompleteJSONContentMessages(t, ReadTextFile(t, path), want...)
}

// AssertCompleteJSONContentMessages 校验 JSON 内容并返回解码结果。
func AssertCompleteJSONContentMessages(
	t *testing.T,
	content string,
	want ...string,
) []map[string]any {
	t.Helper()
	var decoded []map[string]any
	if err := sonic.Unmarshal([]byte(content), &decoded); err != nil {
		t.Fatalf("complete JSON output is invalid: %v\n%s", err, content)
	}
	if len(decoded) != len(want) {
		t.Fatalf(
			"decoded complete JSON has %d events, want %d: %#v",
			len(decoded),
			len(want),
			decoded,
		)
	}
	for index, message := range want {
		if decoded[index]["msg"] != message {
			t.Fatalf(
				"decoded complete JSON event %d msg = %#v, want %q",
				index,
				decoded[index]["msg"],
				message,
			)
		}
	}
	return decoded
}
