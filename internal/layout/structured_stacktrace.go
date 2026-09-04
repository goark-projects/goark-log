package layout

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

func formatStructuredStacktrace(throwable *Throwable, options StructuredStacktraceOptions) string {
	if throwable == nil {
		return ""
	}
	chain := make([]*Throwable, 0, 4)
	for current := throwable; current != nil; current = current.Cause {
		chain = append(chain, current)
		if options.MaxThrowableDepth > 0 && len(chain) >= options.MaxThrowableDepth {
			break
		}
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
		if current.Type != "" {
			builder.WriteString(current.Type)
			builder.WriteString(": ")
		}
		builder.WriteString(current.Message)
		frames := current.Stack
		if !options.IncludeCommonFrames && index > 0 {
			frames = trimCommonFrames(frames, chain[index-1].Stack)
		}
		for _, frame := range frames {
			builder.WriteString("\n\tat ")
			builder.WriteString(frame)
			if options.IncludeHashes {
				builder.WriteString(" #")
				builder.WriteString(strconv.FormatUint(uint64(fnv1a(frame)), 16))
			}
		}
	}
	result := builder.String()
	if options.MaxLength > 0 && len(result) > options.MaxLength {
		result = result[:options.MaxLength]
		for !utf8.ValidString(result) {
			result = result[:len(result)-1]
		}
	}
	return result
}

func trimCommonFrames(frames, parent []string) []string {
	end, parentEnd := len(frames), len(parent)
	for end > 0 && parentEnd > 0 && frames[end-1] == parent[parentEnd-1] {
		end--
		parentEnd--
	}
	return frames[:end]
}

func fnv1a(value string) uint32 {
	const prime = 16777619
	hash := uint32(2166136261)
	for index := 0; index < len(value); index++ {
		hash ^= uint32(value[index])
		hash *= prime
	}
	return hash
}
