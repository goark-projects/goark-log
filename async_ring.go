package goarklog

import "sync"

type asyncRingBuffer struct {
	mu       sync.Mutex
	entries  []asyncLoggerEntry
	head     int
	tail     int
	count    int
	readable chan struct{}
	writable chan struct{}
}

func newAsyncRingBuffer(size int) *asyncRingBuffer {
	return &asyncRingBuffer{
		entries:  make([]asyncLoggerEntry, size),
		readable: make(chan struct{}, 1),
		writable: make(chan struct{}, 1),
	}
}

func (q *asyncRingBuffer) tryPush(entry asyncLoggerEntry) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.count == len(q.entries) {
		return false
	}
	q.entries[q.tail] = entry
	q.tail = (q.tail + 1) % len(q.entries)
	q.count++
	q.signalReadable()
	return true
}

func (q *asyncRingBuffer) popBatch(batch *[]asyncLoggerEntry, max int) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.count == 0 {
		return false
	}
	for q.count > 0 && len(*batch) < max {
		*batch = append(*batch, q.entries[q.head])
		q.entries[q.head] = asyncLoggerEntry{}
		q.head = (q.head + 1) % len(q.entries)
		q.count--
	}
	q.signalWritable()
	return true
}

func (q *asyncRingBuffer) readableSignal() <-chan struct{} {
	return q.readable
}

func (q *asyncRingBuffer) writableSignal() <-chan struct{} {
	return q.writable
}

func (q *asyncRingBuffer) signalReadable() {
	select {
	case q.readable <- struct{}{}:
	default:
	}
}

func (q *asyncRingBuffer) signalWritable() {
	select {
	case q.writable <- struct{}{}:
	default:
	}
}
