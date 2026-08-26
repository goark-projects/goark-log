package goarklog

import (
	"fmt"
	"io/fs"
	"strconv"
	"strings"
)

const defaultLogFilePermissions fs.FileMode = 0o644

func parseLogFilePermissions(value string) (fs.FileMode, error) {
	text := strings.TrimSpace(value)
	if text == "" {
		return defaultLogFilePermissions, nil
	}
	if isOctalFileMode(text) {
		parsed, err := strconv.ParseUint(text, 8, 32)
		if err != nil || parsed > 0o777 {
			return 0, fmt.Errorf("goark-log: filePermissions %q is invalid", value)
		}
		return fs.FileMode(parsed), nil
	}
	mode, ok := parseSymbolicFileMode(text)
	if !ok {
		return 0, fmt.Errorf("goark-log: filePermissions %q is invalid", value)
	}
	return mode, nil
}

func isOctalFileMode(value string) bool {
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

func parseSymbolicFileMode(value string) (fs.FileMode, bool) {
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
		mode |= symbolicFileModeBit(index)
	}
	return mode, true
}

func symbolicFileModeBit(index int) fs.FileMode {
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
