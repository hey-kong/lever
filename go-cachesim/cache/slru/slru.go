package slru

import (
	"container/list"
)

const (
	DefaultProbationRatio = 0.2
)

type LruCache struct {
	MaxEntries int
	OnEvicted  func(key string, value []byte)
	ll         *list.List
	cache      map[interface{}]*list.Element
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

func (c *LruCache) Add(key string, value []byte) (e *list.Element) {
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
		e = c.ll.Back()
		c.RemoveOldest()
	}
	ele := c.ll.PushFront(&entry{key, value})
	c.cache[key] = ele
	return
}

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

func (c *LruCache) Remove(key string) {
	if c.cache == nil {
		return
	}
	if ele, hit := c.cache[key]; hit {
		c.removeElement(ele)
	}
}

func (c *LruCache) RemoveOldest() {
	if c.cache == nil {
		return
	}
	ele := c.ll.Back()
	if ele != nil {
		c.removeElement(ele)
	}
}

func (c *LruCache) removeElement(e *list.Element) {
	c.ll.Remove(e)
	kv := e.Value.(*entry)
	delete(c.cache, kv.key)
	if c.OnEvicted != nil {
		c.OnEvicted(kv.key, kv.value)
	}
}

func (c *LruCache) Len() int {
	if c.cache == nil {
		return 0
	}
	return c.ll.Len()
}

type Cache struct {
	probation *LruCache
	protected *LruCache
}

func NewWithParams(probationSize int, protectedSize int) *Cache {
	return &Cache{
		probation: NewLRU(probationSize),
		protected: NewLRU(protectedSize),
	}
}

func New(size int) *Cache {
	probationSize := int(DefaultProbationRatio * float64(size))
	protectedSize := size - probationSize
	return NewWithParams(
		probationSize,
		protectedSize,
	)
}

func (S *Cache) Add(key string, value []byte) {
	if S.probation == nil || S.protected == nil {
		return
	}

	if e, hit := S.protected.cache[key]; hit {
		S.protected.ll.MoveToFront(e)
		e.Value.(*entry).value = value
		return
	}

	if _, hit := S.probation.cache[key]; hit {
		S.probation.Remove(key)
		if e := S.protected.Add(key, value); e != nil {
			S.probation.Add(e.Value.(*entry).key, e.Value.(*entry).value)
		}
		return
	}

	S.probation.Add(key, value)
}

func (S *Cache) Get(key string) (value []byte, ok bool) {
	if S.probation == nil || S.protected == nil {
		return
	}

	if ele, hit := S.protected.cache[key]; hit {
		S.protected.ll.MoveToFront(ele)
		return ele.Value.(*entry).value, true
	}
	if ele, hit := S.probation.cache[key]; hit {
		S.probation.Remove(key)
		if e := S.protected.Add(key, ele.Value.(*entry).value); e != nil {
			S.probation.Add(e.Value.(*entry).key, e.Value.(*entry).value)
		}
		return ele.Value.(*entry).value, true
	}
	return
}

func (S *Cache) Len() int {
	return S.probation.Len() + S.protected.Len()
}

func (S *Cache) Remove(key string) {
	if _, ok := S.protected.Get(key); ok {
		S.protected.Remove(key)
		return
	}
	S.probation.Remove(key)
}
