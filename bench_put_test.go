package dimcache

import (
	"testing"

	"github.com/hey-kong/dimcache/test/lru"
	"github.com/hey-kong/dimcache/test/sieve"
)

// LRU Put
func BenchmarkLruPutValue64B(b *testing.B) {
	lruCache = lru.New(b.N / 2)
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
	sieveCache = sieve.New(b.N / 2)
	opLen := len(putOperations)

	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		op := putOperations[n%opLen]
		sieveCache.Add(op.key, op.value)
	}
}
