package logfile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DefaultPermissions 是日志文件默认权限。
const DefaultPermissions fs.FileMode = 0o644

// OpenOptions 控制日志文件打开方式。
type OpenOptions struct {
	Append         bool
	Permissions    fs.FileMode
	PermissionsSet bool
}

// ValidatePath 清理并校验日志文件路径。
func ValidatePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("goark-log: log file path is empty")
	}
	cleanPath := filepath.Clean(path)
	info, err := os.Stat(cleanPath)
	if err == nil && info.IsDir() {
		return "", fmt.Errorf("goark-log: log file path %q is a directory", path)
	}
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("goark-log: stat log file %q: %w", path, err)
	}
	return cleanPath, nil
}

// Exists 判断路径是否存在。
func Exists(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else {
		return false, err
	}
}

// Open 按默认追加模式打开日志文件。
func Open(path string) (*os.File, error) {
	return OpenWithOptions(path, OpenOptions{
		Append:         true,
		Permissions:    DefaultPermissions,
		PermissionsSet: true,
	})
}

// OpenWithOptions 按指定选项创建父目录并打开日志文件。
func OpenWithOptions(path string, options OpenOptions) (*os.File, error) {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("goark-log: create log directory %q: %w", dir, err)
		}
	}
	flags := os.O_CREATE | os.O_WRONLY
	if options.Append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	permissions := options.Permissions.Perm()
	if !options.PermissionsSet {
		permissions = DefaultPermissions
	}
	file, err := os.OpenFile(path, flags, permissions)
	if err != nil {
		return nil, fmt.Errorf("goark-log: open log file %q: %w", path, err)
	}
	return file, nil
}

// ParsePermissions 解析八进制或 rwx 符号形式的文件权限。
func ParsePermissions(value string) (fs.FileMode, error) {
	text := strings.TrimSpace(value)
	if text == "" {
		return DefaultPermissions, nil
	}
	if isOctalMode(text) {
		parsed, err := strconv.ParseUint(text, 8, 32)
		if err != nil || parsed > 0o777 {
			return 0, fmt.Errorf("goark-log: filePermissions %q is invalid", value)
		}
		return fs.FileMode(parsed), nil
	}
	mode, ok := parseSymbolicMode(text)
	if !ok {
		return 0, fmt.Errorf("goark-log: filePermissions %q is invalid", value)
	}
	return mode, nil
}

func isOctalMode(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '7' {
			return false
		}
	}
	return true
}

func parseSymbolicMode(value string) (fs.FileMode, bool) {
	if len(value) != 9 {
		return 0, false
	}
	var mode fs.FileMode
	for index, expected := range []byte{'r', 'w', 'x', 'r', 'w', 'x', 'r', 'w', 'x'} {
		char := value[index]
		if char == '-' {
			continue
		}
		if char != expected {
			return 0, false
		}
		mode |= symbolicModeBit(index)
	}
	return mode, true
}

func symbolicModeBit(index int) fs.FileMode {
	switch index {
	case 0:
		return 0o400
	case 1:
		return 0o200
	case 2:
		return 0o100
	case 3:
		return 0o040
	case 4:
		return 0o020
	case 5:
		return 0o010
	case 6:
		return 0o004
	case 7:
		return 0o002
	case 8:
		return 0o001
	default:
		return 0
	}
}
