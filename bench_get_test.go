package dimcache

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
