package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/hey-kong/lever/go-cachesim/cache/fifo"
	refifo "github.com/hey-kong/lever/go-cachesim/cache/fifo_reinsertion"
	"github.com/hey-kong/lever/go-cachesim/cache/lever"
	"github.com/hey-kong/lever/go-cachesim/cache/lru"
	"github.com/hey-kong/lever/go-cachesim/cache/sieve"
	"github.com/hey-kong/lever/go-cachesim/cache/slru"
	twoq "github.com/hey-kong/lever/go-cachesim/cache/twoqueue"
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

// generateTestData generates the unique keys.
func generateTestData(n int) []string {
	keys := make([]string, n)
	for i := 0; i < n; i++ {
		keys[i] = fmt.Sprintf("key%d", i)
	}
	return keys
}

// generateSkewedTestData generates the keys to be accessed based on Zipf distribution.
func generateSkewedTestData(n int, maxKey uint64) []string {
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	zipf := rand.NewZipf(r, 1.07, 1.0, maxKey)

	keys := make([]string, n)
	for i := 0; i < n; i++ {
		keyIndex := zipf.Uint64()
		keys[i] = fmt.Sprintf("key%d", keyIndex)
	}
	return keys
}

// initializeCache initializes cache with a set of keys.
func initializeCache(cache Cache, hotKeys []string) {
	for _, key := range hotKeys {
		cache.Add(key, []byte(fmt.Sprintf("value_%s", key)))
	}
}

// runTest is a generic function to test different cache implementations.
func runTest(cache Cache, cacheName string, keys []string) {
	var hits, misses int
	start := time.Now()

	// Use the pre-generated keys for the test.
	for _, key := range keys {
		_, ok := cache.Get(key)
		if ok {
			hits++
		} else {
			misses++
			cache.Add(key, []byte(key))
		}
	}

	elapsed := time.Since(start)

	fmt.Printf("[%s] Cache Hits: %d, Cache Misses: %d, Hit Rate: %.2f%%\n", cacheName, hits, misses, float64(hits)/float64(hits+misses)*100)
	fmt.Printf("[%s] Total Time: %s, Average Time per Get: %s\n", cacheName, elapsed, elapsed/time.Duration(len(keys)))
}

func main() {
	size := 200000
	lruCache := lru.New(size)
	fifoCache := fifo.New(size)
	refifoCache := refifo.New(size)
	slruCache := slru.New(size)
	twoQCache := twoq.New(size)
	sieveCache := sieve.New(size)
	leverCache := lever.New(size)

	testKeys := generateTestData(size)
	initializeCache(lruCache, testKeys)
	initializeCache(fifoCache, testKeys)
	initializeCache(refifoCache, testKeys)
	initializeCache(slruCache, testKeys)
	initializeCache(twoQCache, testKeys)
	initializeCache(sieveCache, testKeys)
	initializeCache(leverCache, testKeys)

	// Generate a consistent set of test keys for all tests.
	testKeys = generateSkewedTestData(10000000, 999999)
	uniqueKeyCount := countUniqueKeys(testKeys)
	fmt.Printf("Number of testKeys: %d; Number of unique keys in testKeys: %d\n", len(testKeys), uniqueKeyCount)
	// Test each cache.
	runTest(lruCache, "LRU", testKeys)
	runTest(fifoCache, "FIFO", testKeys)
	runTest(refifoCache, "FIFO-Reinsertion", testKeys)
	runTest(slruCache, "SLRU", testKeys)
	runTest(twoQCache, "2Q", testKeys)
	runTest(sieveCache, "SIEVE", testKeys)
	runTest(leverCache, "LEVER", testKeys)
	fmt.Println()

	total, hot := sieveCache.Stats()
	fmt.Printf("SIEVE: Total keys = %d, Hot keys = %d\n", total, hot)
	total, hot = leverCache.Stats()
	fmt.Printf("LEVER: Total keys = %d, Hot keys = %d\n", total, hot)
}
