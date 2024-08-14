package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/hey-kong/lever/cache/fifo"
	"github.com/hey-kong/lever/cache/lever"
	"github.com/hey-kong/lever/cache/lru"
	"github.com/hey-kong/lever/cache/sieve"
)

type Cache interface {
	Add(key string, value []byte)
	Get(key string) (value []byte, ok bool)
}

func countUniqueKeys(keys []string) int {
	keySet := make(map[string]struct{})

	for _, key := range keys {
		keySet[key] = struct{}{}
	}

	return len(keySet)
}

// generateTestData generates the keys to be accessed based on Zipf distribution
func generateTestData(n int, maxKey uint64) []string {
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	zipf := rand.NewZipf(r, 1.01, 1.0, maxKey)

	keys := make([]string, n)
	for i := 0; i < n; i++ {
		keyIndex := zipf.Uint64()
		keys[i] = fmt.Sprintf("key%d", keyIndex)
	}
	return keys
}

// initializeCacheWithHotData initializes cache with a set of "hot" keys
func initializeCacheWithHotData(cache Cache, hotKeys []string) {
	for _, key := range hotKeys {
		cache.Add(key, []byte(fmt.Sprintf("value_%s", key)))
	}
}

// runTest is a generic function to test different cache implementations
func runTest(cache Cache, cacheName string, keys []string) {
	var hits, misses int
	start := time.Now()

	// Use the pre-generated keys for the test
	for _, key := range keys {
		_, ok := cache.Get(key)
		if ok {
			hits++
		} else {
			misses++
			cache.Add(key, []byte(fmt.Sprintf("value_%s", key)))
		}
	}

	elapsed := time.Since(start)

	fmt.Printf("[%s] Cache Hits: %d, Cache Misses: %d, Hit Rate: %.2f%%\n", cacheName, hits, misses, float64(hits)/float64(hits+misses)*100)
	fmt.Printf("[%s] Total Time: %s, Average Time per Get: %s\n", cacheName, elapsed, elapsed/time.Duration(len(keys)))
}

func main() {
	// Generate a consistent set of test keys for all tests
	testKeys := generateTestData(10000000, 999999)
	uniqueKeyCount := countUniqueKeys(testKeys)
	fmt.Printf("Number of testKeys: %d; Number of unique keys in testKeys: %d\n", len(testKeys), uniqueKeyCount)

	lruCache := lru.New(50000)
	fifoCache := fifo.New(50000)
	sieveCache := sieve.New(50000)
	leverCache := lever.New(50000)

	// Test each cache
	runTest(lruCache, "LRU", testKeys)
	runTest(fifoCache, "FIFO", testKeys)
	runTest(sieveCache, "SIEVE", testKeys)
	runTest(leverCache, "LEVER", testKeys)

	total, hot := leverCache.Stats()
	fmt.Printf("[LEVER] Number of keys: %d; Number of hot keys: %d\n", total, hot)
}
