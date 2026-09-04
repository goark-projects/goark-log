package layout

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"goark.dev/log/internal/logevent"
)

func formatStructuredStacktrace(throwable *Throwable, options StructuredStacktraceOptions) string {
	if throwable == nil {
		return ""
	}
	if options.Printer == StructuredStacktracePrinterLoggingSystem {
		return logevent.ThrowableStackString(throwable)
	}
	chain := make([]*Throwable, 0, 4)
	seen := make(map[*Throwable]struct{}, 4)
	var circular *Throwable
	for current := throwable; current != nil; current = current.Cause {
		if _, exists := seen[current]; exists {
			circular = current
			break
		}
		seen[current] = struct{}{}
		chain = append(chain, current)
	}
	if options.RootFirst {
		for left, right := 0, len(chain)-1; left < right; left, right = left+1, right-1 {
			chain[left], chain[right] = chain[right], chain[left]
		}
	}
	var builder strings.Builder
	for index, current := range chain {
		if index > 0 {
			if options.RootFirst {
				builder.WriteString("\nWrapped by: ")
			} else {
				builder.WriteString("\nCaused by: ")
			}
		}
		if options.IncludeHashes {
			appendStackHash(&builder, current)
		}
		if current.Type != "" {
			builder.WriteString(current.Type)
			builder.WriteString(": ")
		}
		builder.WriteString(current.Message)
		enclosing := enclosingThrowable(chain, index, options.RootFirst)
		commonFrames := 0
		if !options.IncludeCommonFrames && enclosing != nil {
			commonFrames = commonFrameCount(current.Stack, enclosing.Stack)
		}
		visibleFrames := len(current.Stack) - commonFrames
		frameLimit := visibleFrames
		if options.MaxThrowableDepth > 0 && frameLimit > options.MaxThrowableDepth {
			frameLimit = options.MaxThrowableDepth
		}
		for _, frame := range current.Stack[:frameLimit] {
			builder.WriteString("\n\tat ")
			builder.WriteString(frame)
		}
		if filtered := visibleFrames - frameLimit; filtered > 0 {
			builder.WriteString("\n\t... ")
			builder.WriteString(strconv.Itoa(filtered))
			builder.WriteString(" filtered")
		}
		if commonFrames > 0 {
			builder.WriteString("\n\t... ")
			builder.WriteString(strconv.Itoa(commonFrames))
			builder.WriteString(" more")
		}
	}
	if circular != nil {
		builder.WriteString("\n[CIRCULAR REFERENCE: ")
		appendThrowableDescription(&builder, circular)
		builder.WriteByte(']')
	}
	result := builder.String()
	if options.MaxLength > 0 && len(result) > options.MaxLength {
		result = truncateStacktrace(result, options.MaxLength)
	}
	return result
}

func appendThrowableDescription(builder *strings.Builder, throwable *Throwable) {
	if throwable.Type != "" {
		builder.WriteString(throwable.Type)
		builder.WriteString(": ")
	}
	builder.WriteString(throwable.Message)
}

func enclosingThrowable(chain []*Throwable, index int, rootFirst bool) *Throwable {
	if rootFirst {
		if index+1 < len(chain) {
			return chain[index+1]
		}
		return nil
	}
	if index > 0 {
		return chain[index-1]
	}
	return nil
}

func truncateStacktrace(value string, maximumLength int) string {
	const ellipsis = "..."
	if maximumLength <= len(ellipsis) {
		return ""
	}
	value = value[:maximumLength-len(ellipsis)]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value + ellipsis
}

func appendStackHash(builder *strings.Builder, throwable *Throwable) {
	hash := uint32(2166136261)
	seen := make(map[*Throwable]struct{}, 4)
	for current := throwable; current != nil; current = current.Cause {
		if _, exists := seen[current]; exists {
			break
		}
		seen[current] = struct{}{}
		hash = fnv1aContinue(hash, current.Type)
		hash = fnv1aContinue(hash, current.Message)
		for _, frame := range current.Stack {
			hash = fnv1aContinue(hash, frame)
		}
	}
	hex := strconv.FormatUint(uint64(hash), 16)
	builder.WriteString("<#")
	for padding := 8 - len(hex); padding > 0; padding-- {
		builder.WriteByte('0')
	}
	builder.WriteString(hex)
	builder.WriteString("> ")
}

func commonFrameCount(frames, parent []string) int {
	end, parentEnd := len(frames), len(parent)
	for end > 0 && parentEnd > 0 && frames[end-1] == parent[parentEnd-1] {
		end--
		parentEnd--
	}
	return len(frames) - end
}

func fnv1aContinue(hash uint32, value string) uint32 {
	const prime = 16777619
	for index := 0; index < len(value); index++ {
		hash ^= uint32(value[index])
		hash *= prime
	}
	return hash
}
