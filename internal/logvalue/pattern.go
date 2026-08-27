package logvalue

import (
	"html"
	"log/slog"
	"strconv"
	"strings"
)

// EncodePatternValue 按 pattern encode 转换器规则转义字符串。
func EncodePatternValue(value string, mode string) string {
	switch mode {
	case "", "json":
		quoted := strconv.Quote(value)
		return strings.TrimSuffix(strings.TrimPrefix(quoted, `"`), `"`)
	case "html":
		return html.EscapeString(value)
	case "xml":
		return html.EscapeString(value)
	case "crlf":
		value = strings.ReplaceAll(value, "\r", `\r`)
		return strings.ReplaceAll(value, "\n", `\n`)
	default:
		return value
	}
}

// HighlightStyle 返回指定级别的默认 ANSI 高亮样式。
func HighlightStyle(level slog.Level, fatalLevel slog.Level) string {
	switch {
	case level >= fatalLevel:
		return "red,bold"
	case level >= slog.LevelError:
		return "red"
	case level >= slog.LevelWarn:
		return "yellow"
	case level >= slog.LevelInfo:
		return "green"
	case level >= slog.LevelDebug:
		return "cyan"
	default:
		return "faint"
	}
}

// ApplyANSIStyle 将 ANSI 样式应用到字符串。
func ApplyANSIStyle(value string, style string) string {
	if value == "" {
		return ""
	}
	start := ansiStyleStart(style)
	if start == "" {
		return value
	}
	return start + value + "\x1b[0m"
}

func ansiStyleStart(style string) string {
	codes := ansiStyleCodes(style)
	if len(codes) == 0 {
		return ""
	}
	return "\x1b[" + strings.Join(codes, ";") + "m"
}

func ansiStyleCodes(style string) []string {
	parts := strings.FieldsFunc(style, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	codes := make([]string, 0, len(parts))
	for _, part := range parts {
		code, ok := ansiStyleCode(part)
		if ok {
			codes = append(codes, code)
		}
	}
	return codes
}

func ansiStyleCode(value string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ReplaceAll(normalized, "_", "")
	switch normalized {
	case "bold":
		return "1", true
	case "dim", "faint":
		return "2", true
	case "underline":
		return "4", true
	case "blink":
		return "5", true
	case "reverse":
		return "7", true
	case "black":
		return "30", true
	case "red":
		return "31", true
	case "green":
		return "32", true
	case "yellow":
		return "33", true
	case "blue":
		return "34", true
	case "magenta", "purple":
		return "35", true
	case "cyan":
		return "36", true
	case "white":
		return "37", true
	case "brightblack", "gray", "grey":
		return "90", true
	case "brightred":
		return "91", true
	case "brightgreen":
		return "92", true
	case "brightyellow":
		return "93", true
	case "brightblue":
		return "94", true
	case "brightmagenta", "brightpurple":
		return "95", true
	case "brightcyan":
		return "96", true
	case "brightwhite":
		return "97", true
	case "bgblack", "backgroundblack":
		return "40", true
	case "bgred", "backgroundred":
		return "41", true
	case "bggreen", "backgroundgreen":
		return "42", true
	case "bgyellow", "backgroundyellow":
		return "43", true
	case "bgblue", "backgroundblue":
		return "44", true
	case "bgmagenta", "bgpurple", "backgroundmagenta", "backgroundpurple":
		return "45", true
	case "bgcyan", "backgroundcyan":
		return "46", true
	case "bgwhite", "backgroundwhite":
		return "47", true
	default:
		return "", false
	}
}

// MaxPatternLength 按 pattern 宽度规则截断字符串。
func MaxPatternLength(value string, limit int) string {
	return truncatePatternWidth(value, limit)
}
