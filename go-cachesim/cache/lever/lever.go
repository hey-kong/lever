package lever

import (
	"container/list"
)

type Cache struct {
	// MaxEntries is the maximum number of cache entries before
	// an item is evicted. Zero means no limit.
	MaxEntries int

	// OnEvicted optionally specifies a callback function to be
	// executed when an entry is purged from the cache.
	OnEvicted func(key string, value []byte)

	fast  *list.Element
	slow  *list.Element
	ll    *list.List
	cache map[interface{}]*list.Element
}

type entry struct {
	key     string
	value   []byte
	visited bool
}

// New creates a new Cache.
// If maxEntries is zero, the cache has no limit and it's assumed
// that eviction is done by the caller.
func New(maxEntries int) *Cache {
	return &Cache{
		MaxEntries: maxEntries,
		fast:       nil,
		slow:       nil,
		ll:         list.New(),
		cache:      make(map[interface{}]*list.Element),
	}
}

// Add adds a value to the cache.
func (c *Cache) Add(key string, value []byte) {
	if c.cache == nil {
		c.cache = make(map[interface{}]*list.Element)
		c.ll = list.New()
		c.fast = nil
		c.slow = nil
	}
	if ee, ok := c.cache[key]; ok {
		ee.Value.(*entry).visited = true
		ee.Value.(*entry).value = value
		return
	}
	if c.MaxEntries != 0 && c.ll.Len() >= c.MaxEntries {
		c.RemoveOldest()
	}
	ele := c.ll.PushFront(&entry{key, value, false})
	c.cache[key] = ele
}

// Get looks up a key's value from the cache.
func (c *Cache) Get(key string) (value []byte, ok bool) {
	if c.cache == nil {
		return
	}
	if ele, hit := c.cache[key]; hit {
		ele.Value.(*entry).visited = true
		return ele.Value.(*entry).value, true
	}
	return
}

// Remove removes the provided key from the cache.
func (c *Cache) Remove(key string) {
	if c.cache == nil {
		return
	}
	if ele, hit := c.cache[key]; hit {
		c.removeElement(ele)
	}
}

// RemoveOldest removes the oldest item from the cache.
func (c *Cache) RemoveOldest() {
	if c.cache == nil {
		return
	}

	if c.slow == nil {
		c.slow = c.ll.Back()
	}
	if c.fast == nil {
		c.fast = c.ll.Back()
	}

	var ele *list.Element
	for i := 0; i < 2; i++ {
		ele, c.fast = c.fast, c.fast.Prev()
		if ele.Value.(*entry).visited {
			ele.Value.(*entry).visited = false
			c.ll.MoveAfter(ele, c.slow)
		}
		if c.fast == nil {
			c.fast = c.ll.Back()
		}
	}

	ele, c.slow = c.slow, c.slow.Prev()
	if ele.Value.(*entry).visited {
		ele.Value.(*entry).visited = false
		// FIFO demotion
		c.ll.Remove(c.ll.Back())
	} else {
		// quick demotion
		c.removeElement(ele)
	}
}

func (c *Cache) removeElement(e *list.Element) {
	if c.fast == e {
		c.fast = c.fast.Prev()
	}
	if c.slow == e {
		c.slow = c.slow.Prev()
	}

	c.ll.Remove(e)
	kv := e.Value.(*entry)
	delete(c.cache, kv.key)
	if c.OnEvicted != nil {
		c.OnEvicted(kv.key, kv.value)
	}
}

// Len returns the number of items in the cache.
func (c *Cache) Len() int {
	if c.cache == nil {
		return 0
	}
	return c.ll.Len()
}

// Clear purges all stored items from the cache.
func (c *Cache) Clear() {
	if c.OnEvicted != nil {
		for _, e := range c.cache {
			kv := e.Value.(*entry)
			c.OnEvicted(kv.key, kv.value)
		}
	}
	c.ll = nil
	c.cache = nil
}

// Stats returns the total number of entries and the number of hot entries.
func (c *Cache) Stats() (total int, hot int) {
	if c.cache == nil {
		return 0, 0
	}

	total = c.ll.Len()
	hot = 0
	for _, e := range c.cache {
		if e.Value.(*entry).visited {
			hot++
		}
	}
	return total, hot
}
