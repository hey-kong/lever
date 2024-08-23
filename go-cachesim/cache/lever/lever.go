package lever

import (
	"container/list"
)

const (
	tempMask  = 1
	visitMask = 1 << tempMask

	DefaultMinHotThreshold = 0.5
)

type Cache struct {
	// MaxEntries is the maximum number of cache entries before
	// an item is evicted. Zero means no limit.
	MaxEntries int

	// OnEvicted optionally specifies a callback function to be
	// executed when an entry is purged from the cache.
	OnEvicted func(key string, value []byte)

	// Number of hot keys.
	hot int

	ptr   *list.Element
	ll    *list.List
	cache map[interface{}]*list.Element
}

type entry struct {
	key    string
	value  []byte
	status uint8
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
		cache:      make(map[interface{}]*list.Element),
	}
}

// Add adds a value to the cache.
func (c *Cache) Add(key string, value []byte) {
	// Initialize the cache and treat the first insertion as hot.
	if c.cache == nil {
		c.ll = list.New()
		c.cache = make(map[interface{}]*list.Element)
		c.ptr = nil
		c.hot = 0
	}

	if ee, ok := c.cache[key]; ok {
		if (ee.Value.(*entry).status & tempMask) == 0 {
			// eager promotion
			c.ll.MoveToFront(ee)
			ee.Value.(*entry).status |= tempMask
			c.hot++
		}
		// non-promotion
		ee.Value.(*entry).status |= visitMask
		ee.Value.(*entry).value = value
		return
	}

	if c.MaxEntries != 0 && c.ll.Len() >= c.MaxEntries {
		c.RemoveOldest()
	}
	ele := c.ll.PushFront(&entry{key, value, tempMask})
	c.cache[key] = ele
	c.hot++
}

// Get looks up a key's value from the cache.
func (c *Cache) Get(key string) (value []byte, ok bool) {
	if c.cache == nil {
		return
	}
	if ele, hit := c.cache[key]; hit {
		if (ele.Value.(*entry).status & tempMask) == 0 {
			// eager promotion
			c.ll.MoveToFront(ele)
			ele.Value.(*entry).status |= tempMask
			c.hot++
		}
		// non-promotion
		ele.Value.(*entry).status |= visitMask
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

	if c.ptr == nil {
		c.ptr = c.ll.Back()
	}

	if (c.ptr.Value.(*entry).status & visitMask) == 0 {
		// quick demotion
		ele := c.ptr
		c.ptr = c.ptr.Prev()
		c.removeElement(ele)
	} else {
		// FIFO demotion
		ele := c.ll.Back()
		c.ptr.Value.(*entry).status = 0
		c.ptr = c.ptr.Prev()
		c.removeElement(ele)
	}

	if float64(c.hot) > float64(c.MaxEntries)*DefaultMinHotThreshold && (c.ptr.Value.(*entry).status&visitMask) != 0 {
		c.ptr.Value.(*entry).status = 0
		c.ptr = c.ptr.Prev()
		c.hot--
	}
}

func (c *Cache) removeElement(e *list.Element) {
	if (c.ptr.Value.(*entry).status & tempMask) != 0 {
		c.hot--
		if c.ptr == e {
			c.ptr = c.ptr.Prev()
		}
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
func (c *Cache) Stats() (int, int) {
	if c.cache == nil {
		return 0, 0
	}
	return c.ll.Len(), c.hot
}
