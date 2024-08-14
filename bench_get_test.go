package lever

import (
	"testing"
)

// LRU Get
func BenchmarkLruGetValue64B(b *testing.B) {
	initLruCache(b.N)
	opLen := len(getOperations)

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		op := getOperations[n%opLen]
		lruCache.Get(op.key)
	}
}

// SIEVE Get
func BenchmarkSieveGetValue64B(b *testing.B) {
	initSieveCache(b.N)
	opLen := len(getOperations)

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		op := getOperations[n%opLen]
		sieveCache.Get(op.key)
	}
}

// FIFO Get
func BenchmarkFifoGetValue64B(b *testing.B) {
	initFifoCache(b.N)
	opLen := len(getOperations)

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		op := getOperations[n%opLen]
		fifoCache.Get(op.key)
	}
}

// LEVER Get
func BenchmarkLeverGetValue64B(b *testing.B) {
	initLeverCache(b.N)
	opLen := len(getOperations)

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		op := getOperations[n%opLen]
		leverCache.Get(op.key)
	}
}
