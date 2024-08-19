package twoqueue

import (
	"container/list"
)

const (
	// Default2QRecentRatio is the ratio of the 2Q cache dedicated
	// to recently added entries that have only been accessed once.
	Default2QRecentRatio = 0.25

	// Default2QGhostEntries is the default ratio of ghost
	// entries kept to track entries recently evicted
	Default2QGhostEntries = 0.5
)

type LruCache struct {
	// MaxEntries is the maximum number of cache entries before
	// an item is evicted. Zero means no limit.
	MaxEntries int

	// OnEvicted optionally specifies a callback function to be
	// executed when an entry is purged from the cache.
	OnEvicted func(key string, value []byte)

	ll    *list.List
	cache map[interface{}]*list.Element
}

type entry struct {
	key   string
	value []byte
}

func NewLRU(maxEntries int) *LruCache {
	return &LruCache{
		MaxEntries: maxEntries,
		ll:         list.New(),
		cache:      make(map[interface{}]*list.Element),
	}
}

// Add adds a value to the cache.
func (c *LruCache) Add(key string, value []byte) {
	if c.cache == nil {
		c.cache = make(map[interface{}]*list.Element)
		c.ll = list.New()
	}
	if ee, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ee)
		ee.Value.(*entry).value = value
		return
	}
	if c.MaxEntries != 0 && c.ll.Len() >= c.MaxEntries {
		c.RemoveOldest()
	}
	ele := c.ll.PushFront(&entry{key, value})
	c.cache[key] = ele
}

// Get looks up a key's value from the cache.
func (c *LruCache) Get(key string) (value []byte, ok bool) {
	if c.cache == nil {
		return
	}
	if ele, hit := c.cache[key]; hit {
		c.ll.MoveToFront(ele)
		return ele.Value.(*entry).value, true
	}
	return
}

// Remove removes the provided key from the cache.
func (c *LruCache) Remove(key string) (ok bool) {
	if c.cache == nil {
		return
	}
	if ele, hit := c.cache[key]; hit {
		c.removeElement(ele)
		return true
	}
	return
}

// RemoveOldest removes the oldest item from the cache.
func (c *LruCache) RemoveOldest() (key string) {
	if c.cache == nil {
		return
	}
	ele := c.ll.Back()
	if ele != nil {
		c.removeElement(ele)
		return ele.Value.(*entry).key
	}
	return
}

func (c *LruCache) removeElement(e *list.Element) {
	c.ll.Remove(e)
	kv := e.Value.(*entry)
	delete(c.cache, kv.key)
	if c.OnEvicted != nil {
		c.OnEvicted(kv.key, kv.value)
	}
}

// Len returns the number of items in the cache.
func (c *LruCache) Len() int {
	if c.cache == nil {
		return 0
	}
	return c.ll.Len()
}

type Cache struct {
	recent      *LruCache // Ain
	frequent    *LruCache // Am
	recentEvict *LruCache // Aout

	capacity            int
	recentCapacity      int
	recentEvictCapacity int
	recentRatio         float64
	ghostRatio          float64
}

// New creates a new instance of TwoQueue with the specified size.
func New(size int) *Cache {
	return NewWithRatio(size, Default2QRecentRatio, Default2QGhostEntries)
}

func NewWithRatio(capacity int, recentRatio, ghostRatio float64) *Cache {
	if capacity <= 0 {
		panic("capacity must be greater than 0")
	}
	if recentRatio < 0.0 || recentRatio > 1.0 {
		panic("recentRatio must be between 0 and 1")
	}
	if ghostRatio < 0.0 || ghostRatio > 1.0 {
		panic("ghostRatio must be between 0 and 1")
	}

	// Determine the sub-capacities
	recentCapacity := int(float64(capacity) * recentRatio)
	recentEvictCapacity := int(float64(capacity) * ghostRatio)

	// Allocate the LRUs
	recent := NewLRU(capacity)
	frequent := NewLRU(capacity)
	recentEvict := NewLRU(recentEvictCapacity)

	return &Cache{
		recent:      recent,
		frequent:    frequent,
		recentEvict: recentEvict,

		capacity:            capacity,
		recentCapacity:      recentCapacity,
		recentEvictCapacity: recentEvictCapacity,
		recentRatio:         recentRatio,
		ghostRatio:          ghostRatio,
	}
}

// Add inserts a key-value pair into the cache.
func (c *Cache) Add(key string, value []byte) {
	// Check if the value is frequently used already,
	// and just update the value
	if _, hit := c.frequent.cache[key]; hit {
		c.frequent.Add(key, value)
		return
	}

	// Check if the value is recently used, and promote
	// the value into the frequent list
	if _, hit := c.recent.cache[key]; hit {
		c.recent.Remove(key)
		c.frequent.Add(key, value)
		return
	}

	// If the value was recently evicted, add it to the
	// frequently used list
	if _, hit := c.recentEvict.cache[key]; hit {
		c.ensureSpace(true)
		c.recentEvict.Remove(key)
		c.frequent.Add(key, value)
		return
	}

	// Add to the recently seen list
	c.ensureSpace(false)
	c.recent.Add(key, value)
}

// Get retrieves the value associated with the key from the cache.
// It returns the value and a boolean indicating whether the key was found.
func (c *Cache) Get(key string) (value []byte, ok bool) {
	if value, ok = c.frequent.Get(key); ok {
		return
	}

	if e, hit := c.recent.cache[key]; hit {
		c.recent.Remove(key)
		c.frequent.Add(key, e.Value.(*entry).value)
		return e.Value.(*entry).value, ok
	}

	return
}

// ensureSpace is used to ensure we have space in the cache
func (c *Cache) ensureSpace(recentEvict bool) {
	// If we have space, nothing to do
	recentLen := c.recent.Len()
	freqLen := c.frequent.Len()
	if recentLen+freqLen < c.capacity {
		return
	}

	// If the recent buffer is larger than
	// the target, evict from there
	if recentLen > 0 && (recentLen > c.recentCapacity || (recentLen == c.recentCapacity && !recentEvict)) {
		key := c.recent.RemoveOldest()
		c.recentEvict.Add(key, []byte{})
		return
	}

	// Remove from the frequent list otherwise
	c.frequent.RemoveOldest()
}

func (c *Cache) Remove(key string) bool {
	return c.frequent.Remove(key) || c.recent.Remove(key) || c.recentEvict.Remove(key)
}

func (c *Cache) Len() int {
	return c.recent.Len() + c.frequent.Len()
}
