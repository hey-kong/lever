[![Go Reference](https://pkg.go.dev/badge/github.com/scalalang2/golang-fifo.svg)](https://pkg.go.dev/github.com/scalalang2/golang-fifo)
[![Go Report Card](https://goreportcard.com/badge/github.com/scalalang2/golang-fifo)](https://goreportcard.com/report/github.com/scalalang2/golang-fifo)
![MIT License](https://img.shields.io/badge/license-MIT-_red.svg)
[![Coverage Status](https://coveralls.io/repos/github/scalalang2/golang-fifo/badge.svg?branch=main)](https://coveralls.io/github/scalalang2/golang-fifo?branch=main)

<h1 align="center">golang-fifo</h1>

This is a modern cache implementation, **inspired** by the following papers, provides high efficiency.

- **SIEVE** | [SIEVE is Simpler than LRU: an Efficient Turn-Key Eviction Algorithm for Web Caches](https://junchengyang.com/publication/nsdi24-SIEVE.pdf) (NSDI'24)
- **S3-FIFO** | [FIFO queues are all you need for cache eviction](https://dl.acm.org/doi/10.1145/3600006.3613147) (SOSP'23)

This offers state-of-the-art efficiency and scalability compared to other LRU-based cache algorithms.

## Basic Usage
```go
import "github.com/scalalang2/golang-fifo/sieve"

size := 1e5
ttl := 0 // 0 means no expiration
cache := sieve.New[string, string](size, ttl)

// set value under hello
cache.Set("hello", "world")

// get value under hello
val, ok := cache.Get("hello")
if ok {
    fmt.Printf("value: %s", val) // => "world"
}

// set more keys
for i := 0; i < 10; i++ {
    cache.Set(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
}

// get number of cache entries
fmt.Printf("len: %d", cache.Len()) // => 11

// remove value under hello
removed := cache.Remove("hello")
if removed {
	fmt.Println("hello was removed")
}
```

## Expiry 
```go
import "github.com/scalalang2/golang-fifo/sieve"

size := 1e5
ttl := time.Second * 3600 // 1 hour
cache := sieve.New[string, string](size, ttl)

// this callback will be called when the element is expired
cahe.SetOnEvict(func(key string, value string) {
    fmt.Printf("key: %s, value: %s was evicted", key, value)
})

// set value under hello
cache.Set("hello", "world")

// remove all cache entries and stop the eviction goroutine.
cache.Close()
```

## Benchmark Result
The benchmark result were obtained using [go-cache-benchmark](https://github.com/scalalang2/go-cache-benchmark)

```
itemSize=500000, workloads=7500000, cacheSize=0.10%, zipf's alpha=0.99, concurrency=16

      CACHE      | HITRATE |   QPS   |  HITS   | MISSES
-----------------+---------+---------+---------+----------
  sieve          | 47.66%  | 2508361 | 3574212 | 3925788
  tinylfu        | 47.37%  | 2269542 | 3552921 | 3947079
  s3-fifo        | 47.17%  | 1651619 | 3538121 | 3961879
  slru           | 46.49%  | 2201350 | 3486476 | 4013524
  s4lru          | 46.09%  | 2484266 | 3456682 | 4043318
  two-queue      | 45.49%  | 1713502 | 3411800 | 4088200
  clock          | 37.34%  | 2370417 | 2800750 | 4699250
  lru-groupcache | 36.59%  | 2206841 | 2743894 | 4756106
  lru-hashicorp  | 36.57%  | 2055358 | 2743000 | 4757000
```

**SIEVE** delivers both high hit rates and the highest QPS(queries per seconds) compared to other LRU-based caches. 
Additionally, It approximately improves 30% for efficiency than a simple LRU cache.

Increasing efficiency means not only reducing cache misses, 
but also reducing the demand for heavy operations such as backend database access, which lowers the mean latency.

While LRU promotes accessed objects to the head of the queue, 
requiring a potentially slow lock acquisition, 
SIEVE only needs to update a single bit upon a cache hit. 
This update can be done with a significantly faster reader lock, leading to increased performance.

The real-world traces are also evaluated at [here](https://observablehq.com/@1a1a11a/sieve-miss-ratio-plots)

## Appendix

<details>
<summary>Performance : golang-fifo</summary>

```shell
goos: linux
goarch: amd64
pkg: github.com/scalalang2/golang-fifo
cpu: Intel(R) Core(TM) i5-10600KF CPU @ 4.10GHz
BenchmarkCache
BenchmarkCache/cache=sieve
BenchmarkCache/cache=sieve/t=int32
BenchmarkCache/cache=sieve/t=int32-12    2765682               393.8 ns/op           148 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=int32-12    3037669               388.1 ns/op           149 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=int32-12    3075998               395.0 ns/op           149 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=int32-12    2924646               392.0 ns/op           148 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=int32-12    2632326               409.3 ns/op           148 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=int32-12    2746551               463.5 ns/op           148 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=int32-12    3004071               401.0 ns/op           148 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=int32-12    2398981               456.0 ns/op           149 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=int32-12    2698939               422.9 ns/op           148 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=int32-12    2647030               392.1 ns/op           148 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=int64
BenchmarkCache/cache=sieve/t=int64-12    2532614               414.1 ns/op           158 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=int64-12    2825973               419.3 ns/op           158 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=int64-12    2693790               407.1 ns/op           158 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=int64-12    2882792               414.7 ns/op           157 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=int64-12    2903197               421.7 ns/op           157 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=int64-12    2876046               435.7 ns/op           157 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=int64-12    2846494               410.4 ns/op           157 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=int64-12    2455807               440.1 ns/op           158 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=int64-12    2774462               435.1 ns/op           158 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=int64-12    2833150               433.9 ns/op           157 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=string
BenchmarkCache/cache=sieve/t=string-12           2117859               546.9 ns/op           186 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=string-12           2079752               527.1 ns/op           186 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=string-12           2210930               530.8 ns/op           186 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=string-12           2122942               514.4 ns/op           186 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=string-12           2222488               553.6 ns/op           186 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=string-12           2260266               558.6 ns/op           186 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=string-12           2239196               567.1 ns/op           186 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=string-12           2064308               576.8 ns/op           186 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=string-12           1882754               569.9 ns/op           185 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=string-12           1917342               574.6 ns/op           185 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=composite
BenchmarkCache/cache=sieve/t=composite-12        1825063               707.0 ns/op           223 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=composite-12        1745775               660.1 ns/op           224 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=composite-12        1680552               678.1 ns/op           225 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=composite-12        1774438               690.1 ns/op           224 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=composite-12        1530580               731.1 ns/op           226 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=composite-12        1663950               761.7 ns/op           225 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=composite-12        1607760               678.4 ns/op           225 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=composite-12        1703283               784.4 ns/op           225 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=composite-12        1295089               864.6 ns/op           229 B/op          4 allocs/op
BenchmarkCache/cache=sieve/t=composite-12        1552182               769.9 ns/op           226 B/op          4 allocs/op
```
</details>

<details>
<summary>Why LRU Cache is not good enough?</summary>

- LRU is often implemented with a doubly linked list and a hash table, requiring two pointers per cache entry,
  which becomes large overhead when the object is small.
- It promotes objects to the head of the queue upon cache hit, which performs at least six random memory accesses
  protected by lock, which limits the scalability.
</details>

<details>
<summary>Things to consider before adoption</summary>

- Both **S3-FIFO** and **SIEVE** have a O(n) time complexity for cache eviction,
  which only occurs when all objects are hit the cache, which means that there is a perfect (100%) hit rate in the cache.
- **SIEVE** is not designed to be scan-resistant. Therefore, it's currently recommended for web cache workloads,
  which typically follow a power-law distribution.
- **S3-FIFO** filters out one-hit-wonders early, It bears some resemblance to designing scan-resistant cache eviction algorithms.
- **SIEVE** scales well for read-intensive applications such as blogs and online shops, because it doesn't require to hold a writer lock on cache hit.
- The `golang-fifo` library aims to provide a straightforward and efficient cache implementation, 
  similar to [hashicorp-lru](https://github.com/hashicorp/golang-lru) and [groupcache](https://github.com/golang/groupcache).
  Its goal is not to outperform highly specialized in-memory cache libraries (e.g. [bigcache](https://github.com/allegro/bigcache), [freecache](https://github.com/coocood/freecache) and etc).
</details>

<details>
<summary>Brief overview of SIEVE & S3-FIFO</summary>

Various workloads typically follows **Power law distribution (e.g. Zipf's law)** as shown in the following figure.

![zipflaw_discovered_by_realworld](./docs/zipf_law_discovered_by_realworld_traces.png)

The analysis reveals that most requests are "one-hit-wonders", which means it's accessed only once.
Consequently, a cache eviction strategy should quickly remove most objects after insertion.

**S3-FIFO** and **SIEVE** achieves this goal with simplicity, efficiency, and scalability using simple FIFO queue only.

![s3-fifo-is-powerful-algorithm](./docs/graphs_shows_s3_fifo_is_powerful.png)
</details>

## Contribution
How to run unit test
```bash
$ go test -v ./...
```

How to run benchmark test
```bash
$ ./bench.sh
```
