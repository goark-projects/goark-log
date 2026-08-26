package disruptor

import "testing"

func BenchmarkRingBufferTryPublishAndPop(b *testing.B) {
	ring, err := NewRingBuffer[int](1024, NewWaitStrategy(WaitYielding))
	if err != nil {
		b.Fatalf("NewRingBuffer() error = %v", err)
	}
	batch := make([]int, 0, 64)
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		for !ring.TryPublish(index) {
			ring.PopBatch(&batch, cap(batch))
			batch = batch[:0]
		}
	}
	for ring.PopBatch(&batch, cap(batch)) {
		batch = batch[:0]
	}
}
