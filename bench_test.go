package lever

import (
	"math/rand"
	"testing"

	"github.com/hey-kong/dimcache/test/fifo"
	"github.com/hey-kong/dimcache/test/lru"
	"github.com/hey-kong/dimcache/test/sieve"
	"github.com/hey-kong/dimcache/util"
)

// CacheInterface defines the interface for a cache
type CacheInterface interface {
	Add(key string, value []byte)
	Get(key string) ([]byte, bool)
}

var (
	getOperations []struct {
		write bool
		key   string
		value []byte
	}
	putOperations []struct {
		write bool
		key   string
		value []byte
	}
	operations []struct {
		write bool
		key   string
		value []byte
	}
)

var lruCache *lru.Cache
var sieveCache *sieve.Cache
var fifoCache *fifo.Cache
var leverCache *Cache

func init() {
	cacheSize := 10000
	numOperations := 1000000
	writeRatio := 0.1
	keyRange := cacheSize * 2

	putOperations = GenerateWorkloadW(numOperations)
	getOperations = GenerateWorkloadR(numOperations)
	operations = GenerateWorkload(numOperations, writeRatio, keyRange)
	initLruCache(cacheSize)
	initSieveCache(cacheSize)
	initFifoCache(cacheSize)
	initLeverCache(cacheSize)
}

func initLruCache(num int) {
	lruCache = lru.New(num)
	for n := 0; n < num; n++ {
		key := util.GetFixedLengthKey(n)
		val, err := util.GetValue(64)
		if err != nil {
			panic(err)
		}
		lruCache.Add(key, val)
	}
}

func initSieveCache(num int) {
	sieveCache = sieve.New(num)
	for n := 0; n < num; n++ {
		key := util.GetFixedLengthKey(n)
		val, err := util.GetValue(64)
		if err != nil {
			panic(err)
		}
		sieveCache.Add(key, val)
	}
}

func initFifoCache(num int) {
	fifoCache = fifo.New(num)
	for n := 0; n < num; n++ {
		key := util.GetFixedLengthKey(n)
		val, err := util.GetValue(64)
		if err != nil {
			panic(err)
		}
		fifoCache.Add(key, val)
	}
}

func initLeverCache(num int) {
	leverCache = New(num)
	for n := 0; n < num; n++ {
		key := util.GetFixedLengthKey(n)
		val, err := util.GetValue(64)
		if err != nil {
			panic(err)
		}
		leverCache.Add(key, val)
	}
}

// GenerateWorkloadW generates a workload of read and write operations
func GenerateWorkloadW(numOperations int) []struct {
	write bool
	key   string
	value []byte
} {
	ops := make([]struct {
		write bool
		key   string
		value []byte
	}, numOperations)

	for i := 0; i < numOperations; i++ {
		tmpKey := util.GetFixedLengthKey(i)
		val, err := util.GetValue(64)
		if err != nil {
			panic(err)
		}
		ops[i] = struct {
			write bool
			key   string
			value []byte
		}{write: true, key: tmpKey, value: val}
	}

	return ops
}

// GenerateWorkloadR generates a workload of read and write operations
func GenerateWorkloadR(numOperations int) []struct {
	write bool
	key   string
	value []byte
} {
	ops := make([]struct {
		write bool
		key   string
		value []byte
	}, numOperations)

	for i := 0; i < numOperations; i++ {
		ops[i] = struct {
			write bool
			key   string
			value []byte
		}{write: false, key: util.GetFixedLengthKey(i), value: util.GetFixedLengthValue(i)}
	}

	return ops
}

// GenerateWorkload generates a workload of read and write operations
func GenerateWorkload(numOperations int, writeRatio float64, keyRange int) []struct {
	write bool
	key   string
	value []byte
} {
	ops := make([]struct {
		write bool
		key   string
		value []byte
	}, numOperations)

	numWrites := int(writeRatio * float64(numOperations))
	for i := 0; i < numWrites; i++ {
		tmpKey := util.GetFixedLengthKey(rand.Intn(keyRange))
		val, err := util.GetValue(64)
		if err != nil {
			panic(err)
		}
		ops[i] = struct {
			write bool
			key   string
			value []byte
		}{write: true, key: tmpKey, value: val}
	}
	for i := numWrites; i < numOperations; i++ {
		ops[i] = struct {
			write bool
			key   string
			value []byte
		}{write: false, key: util.GetFixedLengthKey(rand.Intn(keyRange)), value: nil}
	}

	// Shuffle the operations to ensure random order
	rand.Shuffle(numOperations, func(i, j int) {
		ops[i], ops[j] = ops[j], ops[i]
	})

	return ops
}

// RunBenchmark runs the benchmark on the given cache with the provided workload
func RunBenchmark(b *testing.B, cache CacheInterface) {
	opLen := len(operations)
	for n := 0; n < b.N; n++ {
		op := operations[n%opLen] // Use modulo to cycle through operations
		if op.write {
			cache.Add(op.key, op.value)
		} else {
			cache.Get(op.key)
		}
	}
}

// LRU Hybrid
func BenchmarkLruHybrid(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	RunBenchmark(b, lruCache)
}

// SIEVE Hybrid
func BenchmarkSieveHybrid(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	RunBenchmark(b, sieveCache)
}

// FIFO Hybrid
func BenchmarkFifoHybrid(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	RunBenchmark(b, fifoCache)
}

// LEVER Hybrid
func BenchmarkLeverHybrid(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	RunBenchmark(b, leverCache)
}
