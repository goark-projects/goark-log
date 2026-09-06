package layout

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"goark.dev/log/internal/callsite"
	"goark.dev/log/internal/timepattern"
)

func configurePatternToken(
	token *patternToken,
	converter string,
	options []string,
	layoutOptions LayoutOptions,
) error {
	normalized := strings.ToLower(converter)
	option := firstPatternOption(options)
	switch {
	case converter == "C" || normalized == "class":
		token.kind = tokenCallerClass
	case converter == "M" || normalized == "method":
		token.kind = tokenCallerMethod
	case converter == "F" || normalized == "file":
		token.kind = tokenCallerFile
	case converter == "L" || normalized == "line":
		token.kind = tokenCallerLine
	case converter == "l" || normalized == "location":
		token.kind = tokenCallerLocation
	case normalized == "d" || normalized == "date":
		token.kind = tokenTime
		token.format, token.timeUnix = timepattern.Normalize(option)
	case normalized == "level" || normalized == "p":
		token.kind = tokenLevel
	case normalized == "pid" || normalized == "processid":
		token.kind = tokenPID
	case normalized == "thread" || normalized == "t":
		token.kind = tokenThread
	case normalized == "logger" || converter == "c":
		token.kind = tokenLogger
		token.logger = newLoggerAbbreviator(option)
	case normalized == "msg" || normalized == "message" || converter == "m":
		token.kind = tokenMessage
	case normalized == "attrs" || normalized == "kvp" || normalized == "map":
		token.kind = tokenAttrs
	case converter == "X" || normalized == "mdc":
		if strings.TrimSpace(option) == "" {
			token.kind = tokenAttrs
			return nil
		}
		token.kind = tokenAttr
		token.key = strings.TrimSpace(option)
	case normalized == "ex" || normalized == "throwable" || normalized == "exception":
		token.kind = tokenError
		token.key = strings.ToLower(strings.TrimSpace(option))
	case normalized == "marker":
		token.kind = tokenMarker
	case normalized == "ndc" || normalized == "x":
		token.kind = tokenContextStack
	case normalized == "n":
		token.kind = tokenNewline
	case normalized == "uuid":
		token.kind = tokenUUID
	case normalized == "relative" || normalized == "r":
		token.kind = tokenRelative
	case normalized == "host" || normalized == "hostname":
		token.kind = tokenHost
	case normalized == "sequencenumber" || normalized == "sn":
		token.kind = tokenSequence
	case normalized == "highlight":
		child, err := NewPatternLayoutWithOptions(option, layoutOptions)
		if err != nil {
			return err
		}
		token.kind = tokenHighlight
		token.child = child
	case normalized == "style":
		child, err := NewPatternLayoutWithOptions(option, layoutOptions)
		if err != nil {
			return err
		}
		token.kind = tokenStyle
		token.child = child
		token.value = patternOption(options, 1)
	case normalized == "notempty":
		child, err := NewPatternLayoutWithOptions(option, layoutOptions)
		if err != nil {
			return err
		}
		token.kind = tokenNotEmpty
		token.child = child
	case normalized == "replace":
		if len(options) < 3 {
			return fmt.Errorf(
				"goark-log: replace pattern converter requires pattern, regex and replacement",
			)
		}
		child, err := NewPatternLayoutWithOptions(options[0], layoutOptions)
		if err != nil {
			return err
		}
		expression, err := regexp.Compile(options[1])
		if err != nil {
			return fmt.Errorf("goark-log: replace pattern regex %q is invalid: %w", options[1], err)
		}
		token.kind = tokenReplace
		token.child = child
		token.regex = expression
		token.repl = options[2]
	case normalized == "enc" || normalized == "encode":
		child, err := NewPatternLayoutWithOptions(option, layoutOptions)
		if err != nil {
			return err
		}
		token.kind = tokenEncode
		token.child = child
		token.value = strings.ToLower(strings.TrimSpace(patternOption(options, 1)))
	case normalized == "equals" || normalized == "equalsignorecase":
		if len(options) < 3 {
			return fmt.Errorf(
				"goark-log: %s pattern converter requires pattern, test and substitution",
				converter,
			)
		}
		child, err := NewPatternLayoutWithOptions(options[0], layoutOptions)
		if err != nil {
			return err
		}
		token.kind = tokenEquals
		token.child = child
		token.value = options[1]
		token.repl = options[2]
		token.ignore = normalized == "equalsignorecase"
	case normalized == "maxlen" || normalized == "maxlength":
		if len(options) < 2 {
			return fmt.Errorf("goark-log: maxLen pattern converter requires pattern and length")
		}
		child, err := NewPatternLayoutWithOptions(options[0], layoutOptions)
		if err != nil {
			return err
		}
		limit, err := strconv.Atoi(strings.TrimSpace(options[1]))
		if err != nil || limit < 0 {
			return fmt.Errorf("goark-log: maxLen pattern length %q is invalid", options[1])
		}
		token.kind = tokenMaxLen
		token.child = child
		token.repeat = limit
	case normalized == "repeat":
		if len(options) < 2 {
			return fmt.Errorf("goark-log: repeat pattern converter requires pattern and count")
		}
		child, err := NewPatternLayoutWithOptions(options[0], layoutOptions)
		if err != nil {
			return err
		}
		count, err := strconv.Atoi(strings.TrimSpace(options[1]))
		if err != nil || count < 0 {
			return fmt.Errorf("goark-log: repeat pattern count %q is invalid", options[1])
		}
		token.kind = tokenRepeat
		token.child = child
		token.repeat = count
	default:
		return fmt.Errorf("goark-log: unsupported pattern converter %q", converter)
	}
	return nil
}
func (l *PatternLayout) Format(buf *bytes.Buffer, event Event) error {
	if l == nil {
		return NewDefaultLayout().Format(buf, event)
	}
	var caller callsite.Cache
	for _, token := range l.tokens {
		appendPatternToken(buf, token, event, &caller, l.options)
	}
	return nil
}

// Options 返回布局输出参数快照。
func (l *PatternLayout) Options() LayoutOptions {
	if l == nil {
		return LayoutOptions{}
	}
	return l.options
}

type patternTokenKind int
