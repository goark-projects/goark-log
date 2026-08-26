package disruptor

import (
	"fmt"
	"math/bits"
)

const maxCapacity = 1 << 30

// NormalizeCapacity 把用户容量规整为 Disruptor 友好的 2 的幂。
func NormalizeCapacity(size int) (int, error) {
	if size <= 0 {
		return 0, fmt.Errorf("goark-log: disruptor capacity must be > 0")
	}
	if size > maxCapacity {
		return 0, fmt.Errorf("goark-log: disruptor capacity %d exceeds %d", size, maxCapacity)
	}
	if isPowerOfTwo(size) {
		return size, nil
	}
	return 1 << bits.Len(uint(size)), nil
}

func isPowerOfTwo(size int) bool {
	return size > 0 && size&(size-1) == 0
}
