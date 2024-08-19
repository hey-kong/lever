package fifo_reinsertion

import "container/list"

type Cache struct {
	MaxEntries int
	OnEvicted  func(key string, value []byte)
	ll         *list.List
	cache      map[interface{}]*list.Element
}

type entry struct {
	key     string
	value   []byte
	visited bool
}

func New(maxEntries int) *Cache {
	return &Cache{
		MaxEntries: maxEntries,
		ll:         list.New(),
		cache:      make(map[interface{}]*list.Element),
	}
}

func (c *Cache) Add(key string, value []byte) {
	if c.cache == nil {
		c.cache = make(map[interface{}]*list.Element)
		c.ll = list.New()
	}
	if ee, ok := c.cache[key]; ok {
		ee.Value.(*entry).value = value
		return
	}
	if c.MaxEntries != 0 && c.ll.Len() >= c.MaxEntries {
		c.RemoveOldest()
	}
	ele := c.ll.PushFront(&entry{key, value, false})
	c.cache[key] = ele
}

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
	for ele != nil {
		if ele.Value.(*entry).visited == true {
			prev := ele.Prev()
			c.ll.MoveToFront(ele)
			ele.Value.(*entry).visited = false
			ele = prev
		} else {
			c.removeElement(ele)
			break
		}
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

func (c *Cache) Len() int {
	if c.cache == nil {
		return 0
	}
	return c.ll.Len()
}

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
