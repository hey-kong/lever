package main

import (
	"testing"
)

// LRU Put
func BenchmarkLruPutValue64B(b *testing.B) {
	opLen := len(putOperations)

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		op := putOperations[n%opLen]
		lruCache.Add(op.key, op.value)
	}
}

// SIEVE Put
func BenchmarkSievePutValue64B(b *testing.B) {
	opLen := len(putOperations)

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		op := putOperations[n%opLen]
		sieveCache.Add(op.key, op.value)
	}
}

// FIFO Put
func BenchmarkFifoPutValue64B(b *testing.B) {
	opLen := len(putOperations)

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		op := putOperations[n%opLen]
		fifoCache.Add(op.key, op.value)
	}
}

// LEVER Put
func BenchmarkLeverPutValue64B(b *testing.B) {
	opLen := len(putOperations)

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		op := putOperations[n%opLen]
		leverCache.Add(op.key, op.value)
	}
}
