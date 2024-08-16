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

	// Number of recent moves to the front.
	hot int

	ptr   *list.Element
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
		hot:        0,
		ptr:        nil,
		ll:         list.New(),
		cache:      nil,
	}
}

// Add adds a value to the cache.
func (c *Cache) Add(key string, value []byte) {
	// Initialize the cache and treat the first insertion as hot.
	if c.cache == nil {
		c.ll = list.New()
		c.cache = make(map[interface{}]*list.Element)
		ele := c.ll.PushFront(&entry{key, value, true})
		c.cache[key] = ele
		c.ptr = ele
		c.hot = 1
		return
	}

	if ee, ok := c.cache[key]; ok {
		if ee.Value.(*entry).visited == false {
			c.ll.MoveToFront(ee)
			ee.Value.(*entry).visited = true
			c.hot++
		}
		ee.Value.(*entry).value = value
		return
	}

	ele := c.ll.InsertAfter(&entry{key, value, false}, c.ptr)
	c.cache[key] = ele
	if c.MaxEntries != 0 && c.ll.Len() > c.MaxEntries {
		for c.hot > 99*c.MaxEntries/100 {
			c.ptr.Value.(*entry).visited = false
			c.ptr = c.ptr.Prev()
			c.hot--
		}
		c.RemoveOldest()
	}
}

// Get looks up a key's value from the cache.
func (c *Cache) Get(key string) (value []byte, ok bool) {
	if c.cache == nil {
		return
	}
	if ele, hit := c.cache[key]; hit {
		if ele.Value.(*entry).visited == false {
			c.ll.MoveToFront(ele)
			ele.Value.(*entry).visited = true
			c.hot++
		}
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

func (c *Cache) RemoveOldest() {
	if c.cache == nil {
		return
	}
	ele := c.ll.Back()
	if ele != nil {
		c.removeElement(ele)
	}
}

func (c *Cache) removeElement(e *list.Element) {
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
		if e.Value.(*entry).visited == true {
			hot++
		}
	}
	return total, hot
}
